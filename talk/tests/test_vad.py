"""VAD seam: EnergyVAD.feed(samples) -> Utterance | None, over synthetic PCM."""

import unittest
from array import array

from talk.config import VADConfig
from talk.vad import EnergyVAD

RATE = 24000


def tone(seconds: float, amplitude: float) -> array:
    # Alternating +/-amplitude: RMS == amplitude, no numpy needed.
    n = int(seconds * RATE)
    return array("f", [amplitude if i % 2 else -amplitude for i in range(n)])


def feed_all(vad: EnergyVAD, samples: array, chunk: int = 1024):
    got = []
    for i in range(0, len(samples), chunk):
        u = vad.feed(samples[i : i + chunk])
        if u is not None:
            got.append(u)
    return got


def make_vad(**overrides) -> EnergyVAD:
    return EnergyVAD(VADConfig(**overrides), RATE)


class TestEnergyVAD(unittest.TestCase):
    def test_speech_between_noise_is_one_utterance(self):
        vad = make_vad()
        pcm = tone(1.0, 0.01) + tone(0.5, 0.2) + tone(1.5, 0.01)
        utts = feed_all(vad, pcm)
        self.assertEqual(len(utts), 1)
        u = utts[0]
        # Speech starts at 1.0s; preroll pulls the start a little earlier.
        self.assertAlmostEqual(u.start, 1.0 - 0.3, delta=0.15)
        self.assertGreater(u.end, 1.4)
        self.assertFalse(u.forced_cut)
        # Duration covers the burst plus preroll (hangover tail is trimmed-ish);
        # just require it brackets the speech.
        self.assertGreater(len(u.samples) / RATE, 0.5)

    def test_short_blip_discarded(self):
        vad = make_vad()
        pcm = tone(1.0, 0.01) + tone(0.15, 0.2) + tone(1.5, 0.01)
        self.assertEqual(feed_all(vad, pcm), [])

    def test_pure_noise_never_triggers(self):
        vad = make_vad()
        self.assertEqual(feed_all(vad, tone(5.0, 0.01)), [])

    def test_long_speech_force_cut(self):
        vad = make_vad(max_utterance_seconds=2)
        pcm = tone(1.0, 0.01) + tone(3.0, 0.2)
        utts = feed_all(vad, pcm)
        self.assertGreaterEqual(len(utts), 1)
        self.assertTrue(utts[0].forced_cut)
        self.assertAlmostEqual(len(utts[0].samples) / RATE, 2.0, delta=0.4)

    def test_inter_word_pause_does_not_split(self):
        vad = make_vad()
        # 300ms pause is shorter than the 800ms hangover: one utterance.
        pcm = (
            tone(1.0, 0.01)
            + tone(0.5, 0.2)
            + tone(0.3, 0.01)
            + tone(0.5, 0.2)
            + tone(1.5, 0.01)
        )
        utts = feed_all(vad, pcm)
        self.assertEqual(len(utts), 1)

    def test_two_utterances_split_by_long_silence(self):
        vad = make_vad()
        pcm = (
            tone(1.0, 0.01)
            + tone(0.5, 0.2)
            + tone(2.0, 0.01)
            + tone(0.5, 0.2)
            + tone(1.5, 0.01)
        )
        utts = feed_all(vad, pcm)
        self.assertEqual(len(utts), 2)
        self.assertGreater(utts[1].start, utts[0].end)


if __name__ == "__main__":
    unittest.main()
