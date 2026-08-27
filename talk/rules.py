"""Canned replies for standard ham exchanges.

Regex hit = high confidence, so routing (in brain.py) is simply "rules
first, Claude only when no rule matches" — no separate confidence score
needed here.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

from talk.config import ScriptsConfig


@dataclass(frozen=True)
class RuleContext:
    last_reply_text: str | None = None


@dataclass(frozen=True)
class RuleReply:
    text: str
    intent: str


_PATTERNS = {
    "signal_report": re.compile(r"signal report|how('?s| is)? my (signal|audio)|how (do you |'?)copy"),
    "presence": re.compile(r"(are\s+)?you\s+(still\s+)?there|you\s+copy\b"),
    "name_qth": re.compile(r"your\s+name|\bq\.?\s*t\.?\s*h\b|your\s+location|where\s+are\s+you"),
    "repeat": re.compile(r"say\s+again|repeat(\s+that|\s+your\s+last)?|again\s+please"),
}


class RuleEngine:
    def __init__(self, scripts: ScriptsConfig, operator_name: str = "", qth: str = ""):
        self._scripts = scripts
        self._operator_name = operator_name
        self._qth = qth

    def reply(self, text: str, ctx: RuleContext) -> RuleReply | None:
        text = text.lower()
        for intent, pattern in _PATTERNS.items():
            if not pattern.search(text):
                continue
            if intent == "signal_report":
                return RuleReply(self._scripts.signal_report_template, intent)
            if intent == "presence":
                return RuleReply("Yes, I'm here, go ahead.", intent)
            if intent == "name_qth":
                bits = []
                if self._operator_name:
                    bits.append(f"the operator is {self._operator_name}")
                if self._qth:
                    bits.append(f"QTH is {self._qth}")
                return RuleReply(", ".join(bits) or "No name or QTH configured.", intent)
            if intent == "repeat":
                if ctx.last_reply_text:
                    return RuleReply(ctx.last_reply_text, intent)
                continue  # nothing to repeat yet; let another rule/Claude try
        return None
