"""Matcher seam: match(text, station) -> TriggerMatch."""

import unittest

from talk.config import StationConfig
from talk.matcher import match

STATION = StationConfig(
    callsign="W5TSU",
    phonetic_words=("whiskey", "five", "tango", "sierra", "uniform"),
    wake_names=("thetis",),
)


class TestPhoneticCallsign(unittest.TestCase):
    def test_full_callsign_triggers(self):
        m = match("calling whiskey five tango sierra uniform are you there", STATION)
        self.assertTrue(m.triggered)
        self.assertEqual(m.kind, "callsign")

    def test_three_of_five_in_order_triggers(self):
        m = match("whiskey five tango do you copy", STATION)
        self.assertTrue(m.triggered)

    def test_misheard_words_still_trigger(self):
        # Whisper-style mangling: whisky/fife variants within fuzzy tolerance.
        m = match("whisky fife tango sierra uniform", STATION)
        self.assertTrue(m.triggered)

    def test_two_of_five_does_not_trigger(self):
        m = match("whiskey five is my drink order", STATION)
        self.assertFalse(m.triggered)

    def test_unrelated_speech_is_silent(self):
        m = match("the quick brown fox jumps over the lazy dog", STATION)
        self.assertFalse(m.triggered)

    def test_words_spread_beyond_window_do_not_trigger(self):
        # 3 matching words but scattered across far more than a 7-token window.
        text = "whiskey is nice and so is a number like five but mostly I dance the tango"
        self.assertFalse(match(text, STATION).triggered)

    def test_later_fuzzy_match_does_not_block_earlier_words(self):
        # "give" ~ "five" at exactly WORD_RATIO; a greedy scan matches it,
        # advances past tango/uniform, and drops to 2 hits. The real callsign
        # words are all present and in order.
        m = match("whiskey tango uniform please give me your location over", STATION)
        self.assertTrue(m.triggered)
        self.assertEqual(m.kind, "callsign")

    def test_full_callsign_with_give_in_the_message_triggers(self):
        m = match(
            "whiskey five tango sierra uniform please give me your location over",
            STATION,
        )
        self.assertTrue(m.triggered)

    def test_empty_text(self):
        self.assertFalse(match("", STATION).triggered)


class TestWakeNames(unittest.TestCase):
    def test_wake_name_triggers(self):
        m = match("hey thetis what time is it", STATION)
        self.assertTrue(m.triggered)
        self.assertEqual(m.kind, "wake")

    def test_close_misspelling_triggers(self):
        self.assertTrue(match("hey thetiss are you on", STATION).triggered)

    def test_distant_word_does_not(self):
        self.assertFalse(match("the tesla is charging", STATION).triggered)


if __name__ == "__main__":
    unittest.main()
