"""Resampler (always) and the real-Whisper wrapper (opt-in via TALK_MODELS_DIR)."""

import math
import os
import unittest
from array import array

from talk.transcribe import resample

RATE = 24000


def sine(freq, seconds, rate):
    return array(
        "f", (math.sin(2 * math.pi * freq * i / rate) for i in range(int(seconds * rate)))
    )


def zero_crossings(samples):
    return sum(
        1 for a, b in zip(samples, samples[1:]) if (a < 0) != (b < 0)
    )


class TestResample(unittest.TestCase):
    def test_24k_to_16k_length(self):
        out = resample(sine(440, 1.0, 24000), 24000, 16000)
        self.assertEqual(len(out), 16000)

    def test_tone_frequency_preserved(self):
        src = sine(440, 1.0, 24000)
        out = resample(src, 24000, 16000)
        # 440 Hz has ~880 zero crossings per second at any adequate rate.
        self.assertAlmostEqual(zero_crossings(out), 880, delta=6)

    def test_identity_when_rates_equal(self):
        src = sine(200, 0.1, 16000)
        self.assertEqual(list(resample(src, 16000, 16000)), list(src))

    def test_48k_to_16k(self):
        out = resample(sine(440, 0.5, 48000), 48000, 16000)
        self.assertEqual(len(out), 8000)
        self.assertAlmostEqual(zero_crossings(out), 440, delta=6)


@unittest.skipUnless(os.environ.get("TALK_MODELS_DIR"), "real-model test; set TALK_MODELS_DIR")
class TestRealWhisper(unittest.TestCase):
    def test_silence_transcribes_to_nothing_much(self):
        from talk.config import VADConfig  # noqa: F401  (env sanity)
        from talk.transcribe import Transcriber

        t = Transcriber(models_dir=os.environ["TALK_MODELS_DIR"])
        text = t.transcribe(array("f", [0.0] * RATE), RATE).text
        self.assertLess(len(text.split()), 4)


if __name__ == "__main__":
    unittest.main()
