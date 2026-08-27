"""SessionLog seam: per-session dir with WAVs and a JSONL transcript."""

import json
import pathlib
import tempfile
import unittest
import wave
from array import array

from talk.qsolog import SessionLog


class TestSessionLog(unittest.TestCase):
    def test_records_wavs_and_jsonl(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            samples = array("f", [0.0, 0.5, -0.5, 0.25])
            wav_path = log.write_wav("rx-utterance", samples, 24000)
            log.event("utterance", transcript="test one two", wav=str(wav_path))
            log.event("decision", source="rule", reply="hello")
            log.close()

            session_dir = pathlib.Path(wav_path).parent
            self.assertTrue(session_dir.is_dir())
            with wave.open(str(wav_path)) as w:
                self.assertEqual(w.getframerate(), 24000)
                self.assertEqual(w.getnchannels(), 1)
                self.assertEqual(w.getsampwidth(), 2)
                self.assertEqual(w.getnframes(), 4)

            lines = (session_dir / "session.jsonl").read_text().splitlines()
            records = [json.loads(l) for l in lines]
            kinds = [r["event"] for r in records]
            self.assertIn("utterance", kinds)
            self.assertIn("decision", kinds)
            for r in records:
                self.assertIn("t", r)

    def test_disabled_writes_nothing(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=False)
            p = log.write_wav("rx-utterance", array("f", [0.1]), 24000)
            log.event("utterance", transcript="x")
            log.close()
            self.assertIsNone(p)
            self.assertEqual(list(pathlib.Path(tmp).iterdir()), [])


if __name__ == "__main__":
    unittest.main()
