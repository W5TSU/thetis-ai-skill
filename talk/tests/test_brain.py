"""Brain seam: compose(heard, qso) -> Decision(reply_text, source, intent).

Routing is rules-first: Claude is consulted only when no rule matches.
"""

import unittest
from dataclasses import dataclass, field

from talk.brain import Brain, QsoContext
from talk.config import ClaudeConfig, ScriptsConfig, StationConfig
from talk.rules import RuleEngine

try:
    import anthropic

    HAS_ANTHROPIC = True
except ImportError:
    HAS_ANTHROPIC = False

SCRIPTS = ScriptsConfig(
    id_text="id",
    signoff="clear",
    fallback_reply="Please say again.",
    signal_report_template="Good copy on your signal.",
)
STATION = StationConfig(
    callsign="W5TSU",
    phonetic_words=("whiskey", "five", "tango", "sierra", "uniform"),
    operator_name="Mark",
    qth="Oklahoma City",
)


@dataclass
class FakeMessage:
    text: str


@dataclass
class FakeResponse:
    content: list


class FakeMessagesAPI:
    """Stand-in for client.messages — records calls, plays back a script."""

    def __init__(self, responses=None, error=None):
        self.responses = list(responses or [])
        self.error = error
        self.calls = []

    def create(self, **kwargs):
        self.calls.append(kwargs)
        if self.error is not None:
            raise self.error
        text = self.responses.pop(0) if self.responses else "..."
        return FakeResponse(content=[FakeMessage(text=text)])


class FakeClient:
    def __init__(self, **kwargs):
        self.messages = kwargs.pop("messages_api")


def make_brain(client=None):
    return Brain(
        rule_engine=RuleEngine(SCRIPTS, STATION.operator_name, STATION.qth),
        scripts=SCRIPTS,
        station=STATION,
        claude_config=ClaudeConfig(),
        claude_client=client,
    )


class TestRulesFirst(unittest.TestCase):
    def test_rule_match_never_calls_claude(self):
        api = FakeMessagesAPI()
        client = FakeClient(messages_api=api)
        brain = make_brain(client)
        decision = brain.compose("how do you copy", QsoContext())
        self.assertEqual(decision.source, "rule")
        self.assertEqual(decision.reply_text, "Good copy on your signal.")
        self.assertEqual(api.calls, [])


@unittest.skipUnless(HAS_ANTHROPIC, "needs the anthropic package installed")
class TestClaudeFallthrough(unittest.TestCase):
    def test_no_rule_match_calls_claude(self):
        api = FakeMessagesAPI(responses=["Sure, the weather here is clear."])
        brain = make_brain(FakeClient(messages_api=api))
        decision = brain.compose("what's the weather like", QsoContext())
        self.assertEqual(decision.source, "claude")
        self.assertEqual(decision.reply_text, "Sure, the weather here is clear.")
        self.assertEqual(len(api.calls), 1)
        self.assertIn("system", api.calls[0])
        self.assertIn("automated", api.calls[0]["system"].lower())

    def test_qso_context_sent_as_message_history(self):
        api = FakeMessagesAPI(responses=["reply two"])
        brain = make_brain(FakeClient(messages_api=api))
        qso = QsoContext()
        qso.add_turn("first thing heard", "reply one")
        brain.compose("second thing heard", qso)
        messages = api.calls[0]["messages"]
        contents = [m["content"] for m in messages]
        self.assertIn("first thing heard", contents)
        self.assertIn("reply one", contents)
        self.assertIn("second thing heard", contents)

    def test_context_window_caps_at_max_turns(self):
        api = FakeMessagesAPI(responses=["latest reply"])
        brain = Brain(
            rule_engine=RuleEngine(SCRIPTS, STATION.operator_name, STATION.qth),
            scripts=SCRIPTS,
            station=STATION,
            claude_config=ClaudeConfig(max_context_turns=2),
            claude_client=FakeClient(messages_api=api),
        )
        qso = QsoContext()
        for i in range(5):
            qso.add_turn(f"heard {i}", f"reply {i}")
        brain.compose("heard latest", qso)
        messages = api.calls[0]["messages"]
        # 2 turns kept = 4 messages (heard/reply pairs) + the new heard message.
        self.assertEqual(len(messages), 5)


class TestDegradation(unittest.TestCase):
    def test_no_client_configured_uses_fallback(self):
        brain = make_brain(client=None)
        decision = brain.compose("what's the weather like", QsoContext())
        self.assertEqual(decision.source, "fallback")
        self.assertEqual(decision.reply_text, SCRIPTS.fallback_reply)

    @unittest.skipUnless(HAS_ANTHROPIC, "needs the anthropic package installed")
    def test_api_error_falls_back(self):
        api = FakeMessagesAPI(error=anthropic.APIConnectionError(request=None))
        brain = make_brain(FakeClient(messages_api=api))
        decision = brain.compose("what's the weather like", QsoContext())
        self.assertEqual(decision.source, "fallback")
        self.assertEqual(decision.reply_text, SCRIPTS.fallback_reply)

    @unittest.skipUnless(HAS_ANTHROPIC, "needs the anthropic package installed")
    def test_timeout_falls_back(self):
        api = FakeMessagesAPI(error=anthropic.APITimeoutError(request=None))
        brain = make_brain(FakeClient(messages_api=api))
        decision = brain.compose("what's the weather like", QsoContext())
        self.assertEqual(decision.source, "fallback")


if __name__ == "__main__":
    unittest.main()
