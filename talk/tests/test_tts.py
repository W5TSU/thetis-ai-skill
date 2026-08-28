"""Synthesizer seam: synthesize(text, out_path) -> duration seconds.

Output must always be 48000 Hz / 1 channel / 16-bit — the only WAV format
proven against real TX hardware.
"""

import os
import pathlib
import tempfile
import unittest
import wave
from array import array

from talk.tts import (
    LEAD_SILENCE_MS,
    TRAIL_SILENCE_MS,
    TX_RATE,
    _with_silence,
    drop_trailing_sentences,
    synthesize_capped,
)


class FakeSynth:
    """1.0s per word, so callers can predict duration from text length."""

    def __init__(self):
        self.calls = []

    def synthesize(self, text, out_path):
        self.calls.append(text)
        return float(len(text.split()))


class TestSynthesizeCapped(unittest.TestCase):
    def test_under_budget_synthesizes_once(self):
        synth = FakeSynth()
        duration, refused, final_text = synthesize_capped(synth, "one two three", "/tmp/x.wav", 60)
        self.assertEqual(duration, 3.0)
        self.assertFalse(refused)
        self.assertEqual(final_text, "one two three")
        self.assertEqual(len(synth.calls), 1)

    def test_over_budget_drops_sentences_and_resynthesizes(self):
        synth = FakeSynth()
        text = "one two three. four five six seven eight nine ten eleven twelve."
        duration, refused, final_text = synthesize_capped(synth, text, "/tmp/x.wav", 5)
        self.assertFalse(refused)
        self.assertLessEqual(duration, 5)
        self.assertIn("one two three.", final_text)
        self.assertNotIn("twelve", final_text)

    def test_single_sentence_over_budget_is_refused(self):
        synth = FakeSynth()
        text = "one two three four five six seven eight nine ten eleven twelve."
        duration, refused, final_text = synthesize_capped(synth, text, "/tmp/x.wav", 5)
        self.assertTrue(refused)


class TestSilencePadding(unittest.TestCase):
    def test_brackets_pcm_with_lead_and_trail_silence(self):
        body = array("h", [6000, -6000, 6000, -6000])
        out = _with_silence(body)
        n_lead = round(TX_RATE * LEAD_SILENCE_MS / 1000)
        n_trail = round(TX_RATE * TRAIL_SILENCE_MS / 1000)
        self.assertEqual(len(out), n_lead + len(body) + n_trail)
        self.assertEqual(set(out[:n_lead]), {0})
        self.assertEqual(set(out[-n_trail:]), {0})
        self.assertEqual(list(out[n_lead : n_lead + len(body)]), list(body))

    def test_empty_body_is_all_silence(self):
        out = _with_silence(array("h"))
        self.assertGreater(len(out), 0)
        self.assertEqual(set(out), {0})


class TestTruncationHelper(unittest.TestCase):
    def test_drops_last_sentence_when_over_budget(self):
        text = "First sentence here. Second sentence here. Third sentence here."
        # Assume ~3 words/sec spoken; budget forces at least one drop.
        kept = drop_trailing_sentences(text, seconds_per_sentence_estimate=lambda s: 5.0, budget_seconds=12.0)
        self.assertIn("First sentence here.", kept)
        self.assertNotIn("Third sentence here.", kept)

    def test_single_sentence_over_budget_is_unchanged(self):
        text = "One very long unbreakable sentence with no other parts at all."
        kept = drop_trailing_sentences(text, seconds_per_sentence_estimate=lambda s: 999.0, budget_seconds=5.0)
        self.assertEqual(kept, text)

    def test_already_under_budget_is_unchanged(self):
        text = "Short reply."
        kept = drop_trailing_sentences(text, seconds_per_sentence_estimate=lambda s: 1.0, budget_seconds=60.0)
        self.assertEqual(kept, text)


@unittest.skipUnless(os.environ.get("TALK_MODELS_DIR"), "real-model test; set TALK_MODELS_DIR")
class TestRealPiper(unittest.TestCase):
    def test_synthesize_produces_tx_ready_wav(self):
        from talk.tts import Synthesizer

        synth = Synthesizer(
            model_path=os.environ["TALK_MODELS_DIR"] + "/piper/en_US-lessac-medium.onnx"
        )
        with tempfile.TemporaryDirectory() as tmp:
            out = pathlib.Path(tmp) / "reply.wav"
            duration = synth.synthesize("This is a test transmission.", out)
            self.assertGreater(duration, 0.3)
            with wave.open(str(out)) as w:
                self.assertEqual(w.getframerate(), TX_RATE)
                self.assertEqual(w.getnchannels(), 1)
                self.assertEqual(w.getsampwidth(), 2)
                self.assertAlmostEqual(w.getnframes() / TX_RATE, duration, delta=0.2)
                frames = array("h")
                frames.frombytes(w.readframes(w.getnframes()))
            # Silence brackets the speech so ALSA/PTT don't clip a syllable.
            head = round(TX_RATE * LEAD_SILENCE_MS / 1000)
            tail = round(TX_RATE * TRAIL_SILENCE_MS / 1000)
            self.assertEqual(set(frames[: head // 2]), {0})
            self.assertEqual(set(frames[-(tail // 2) :]), {0})
            self.assertTrue(any(abs(s) > 100 for s in frames))  # speech present


if __name__ == "__main__":
    unittest.main()
