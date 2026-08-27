"""End-to-end at the CLI seam: fake stream in, utterance records out."""

import json
import pathlib
import subprocess
import sys
import tempfile
import unittest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
FAKE_STREAM = REPO_ROOT / "talk" / "tests" / "fake_stream.py"

CONFIG_TEMPLATE = """
[radio]
host = "test-invalid-host"

[station]
callsign = "W5TSU"
phonetic_words = ["whiskey", "five", "tango", "sierra", "uniform"]

[scripts]
id_text = "This is W5TSU."
signoff = "W5TSU clear."

[logging]
dir = "{log_dir}"
"""


class TestListenE2E(unittest.TestCase):
    def test_fake_stream_produces_utterance_records_then_dead_exit(self):
        with tempfile.TemporaryDirectory() as tmp:
            log_dir = pathlib.Path(tmp) / "logs"
            cfg = pathlib.Path(tmp) / "config.toml"
            cfg.write_text(CONFIG_TEMPLATE.format(log_dir=log_dir))
            result = subprocess.run(
                [sys.executable, "-m", "talk", "--config", str(cfg)],
                cwd=REPO_ROOT,
                capture_output=True,
                text=True,
                timeout=60,
                env={
                    "PATH": "/usr/bin:/bin",
                    "TALK_STREAM_CMD": f"{sys.executable} {FAKE_STREAM} 1",
                },
            )
            # Fake child exits 1 = radio died: the loop reports and exits nonzero.
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

            sessions = list(log_dir.iterdir())
            self.assertEqual(len(sessions), 1)
            records = [
                json.loads(l)
                for l in (sessions[0] / "session.jsonl").read_text().splitlines()
            ]
            kinds = [r["event"] for r in records]
            self.assertIn("utterance", kinds)
            self.assertIn("rx-dead", kinds)
            utt = next(r for r in records if r["event"] == "utterance")
            wav = pathlib.Path(utt["wav"])
            self.assertTrue(wav.exists())
            # The 0.8s burst plus 0.3s preroll and 0.8s hangover tail.
            self.assertGreaterEqual(utt["duration"], 0.8)
            self.assertLessEqual(utt["duration"], 2.5)
            # Live supervision line reached the terminal.
            self.assertIn("utterance", result.stdout.lower())


if __name__ == "__main__":
    unittest.main()
