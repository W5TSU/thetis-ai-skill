"""RuleEngine seam: reply(text, ctx) -> str | None."""

import unittest

from talk.config import ScriptsConfig
from talk.rules import RuleEngine, RuleContext

SCRIPTS = ScriptsConfig(
    id_text="id",
    signoff="clear",
    signal_report_template="Good copy on your signal.",
)


class TestRuleEngine(unittest.TestCase):
    def setUp(self):
        self.engine = RuleEngine(SCRIPTS, operator_name="Mark", qth="Oklahoma City")

    def test_signal_report_request(self):
        for text in ["how do you copy", "how's my signal", "signal report please"]:
            reply = self.engine.reply(text, RuleContext())
            self.assertEqual(reply.text, SCRIPTS.signal_report_template)
            self.assertEqual(reply.intent, "signal_report")

    def test_presence_check(self):
        reply = self.engine.reply("are you still there", RuleContext())
        self.assertIsNotNone(reply)
        self.assertEqual(reply.intent, "presence")

    def test_name_qth(self):
        reply = self.engine.reply("what's your qth", RuleContext())
        self.assertIn("Oklahoma City", reply.text)
        reply2 = self.engine.reply("what is your name", RuleContext())
        self.assertIn("Mark", reply2.text)

    def test_repeat_resends_last_reply_verbatim(self):
        ctx = RuleContext(last_reply_text="Good copy on your signal.")
        reply = self.engine.reply("say again please", ctx)
        self.assertEqual(reply.text, "Good copy on your signal.")
        self.assertEqual(reply.intent, "repeat")

    def test_repeat_with_no_prior_reply_does_not_match(self):
        reply = self.engine.reply("say again please", RuleContext())
        self.assertIsNone(reply)

    def test_no_rule_matches_conversational_text(self):
        reply = self.engine.reply("what's the weather like where you are", RuleContext())
        self.assertIsNone(reply)


if __name__ == "__main__":
    unittest.main()
