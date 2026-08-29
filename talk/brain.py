"""Compose replies: canned rules first, Claude for everything else, a
scripted fallback if neither is available or Claude fails.

The persona and hard rules are the same regardless of how conversational
the reply needs to be: identify as automated if asked, keep it brief, plain
language, no third-party traffic, no promises to change frequency or mode.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from talk.config import ClaudeConfig, ScriptsConfig, StationConfig
from talk.rules import RuleContext, RuleEngine

HARD_RULES = (
    "You are an automated voice station operating under the supervision of "
    "a licensed radio operator. If asked, identify yourself as automated. "
    "Keep every reply under about 45 seconds spoken (roughly 100 words). "
    "Use plain conversational language, not jargon dumps. Never pass "
    "third-party traffic or relay messages for anyone. Never promise to "
    "change frequency or mode. Use standard phonetics when spelling."
)

MAX_TOKENS = 300


@dataclass
class Decision:
    reply_text: str
    source: str  # "rule" | "claude" | "fallback"
    intent: str | None = None


@dataclass
class QsoContext:
    """Turns exchanged in the current QSO. Discarded when the QSO closes."""

    turns: list[tuple[str, str]] = field(default_factory=list)

    def add_turn(self, heard: str, reply: str) -> None:
        self.turns.append((heard, reply))


def _system_prompt(station: StationConfig) -> str:
    persona = (
        f"You are the automated voice assistant for station {station.callsign}"
        + (f", operated by {station.operator_name}" if station.operator_name else "")
        + (f", located at {station.qth}" if station.qth else "")
        + "."
    )
    return f"{persona}\n\n{HARD_RULES}"


class Brain:
    def __init__(
        self,
        rule_engine: RuleEngine,
        scripts: ScriptsConfig,
        station: StationConfig,
        claude_config: ClaudeConfig,
        claude_client=None,
    ):
        self._rules = rule_engine
        self._scripts = scripts
        self._station = station
        self._claude_config = claude_config
        self._client = claude_client
        self._last_reply_text: str | None = None

    def compose(self, heard: str, qso: QsoContext) -> Decision:
        decision = self._decide(heard, qso)
        self._last_reply_text = decision.reply_text
        # Every reply becomes part of the QSO's history, not just Claude's —
        # a later Claude turn should see an earlier rule/fallback reply too,
        # and it's this call's job to grow the context the caller passed in
        # (nothing else populates it).
        qso.add_turn(heard, decision.reply_text)
        return decision

    def _decide(self, heard: str, qso: QsoContext) -> Decision:
        rule_reply = self._rules.reply(heard, RuleContext(last_reply_text=self._last_reply_text))
        if rule_reply is not None:
            return Decision(rule_reply.text, "rule", rule_reply.intent)

        if self._client is not None:
            text = self._ask_claude(heard, qso)
            if text is not None:
                return Decision(text, "claude")

        return Decision(self._scripts.fallback_reply, "fallback")

    def _ask_claude(self, heard: str, qso: QsoContext) -> str | None:
        import anthropic

        messages = []
        kept = qso.turns[-self._claude_config.max_context_turns :]
        for prior_heard, prior_reply in kept:
            messages.append({"role": "user", "content": prior_heard})
            messages.append({"role": "assistant", "content": prior_reply})
        messages.append({"role": "user", "content": heard})

        try:
            response = self._client.messages.create(
                model=self._claude_config.model,
                max_tokens=MAX_TOKENS,
                system=_system_prompt(self._station),
                messages=messages,
                timeout=self._claude_config.timeout_seconds,
            )
        except anthropic.APIError:
            return None
        # Claude 5 models lead with a thinking block; take the first text
        # block, not content[0]. No text block at all -> fall back.
        text = next(
            (b.text for b in response.content if getattr(b, "type", None) == "text"),
            None,
        )
        return text.strip() if text else None
