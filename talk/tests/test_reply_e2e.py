"""End-to-end at the CLI seam with real models: synthesized speech in via
TALK_STREAM_CMD, a `reply` (or no reply) out.

The sibling test_listen_e2e.py stops at "an utterance was logged" because
its fake stream is a bare tone. This one feeds real Piper speech so the
Whisper -> matcher -> rules -> Piper -> player path actually runs, which is
why it needs the models and is opt-in.

    TALK_MODELS_DIR=talk/models talk/.venv/bin/python -m unittest talk.tests.test_reply_e2e
"""

import json
import os
import pathlib
import shlex
import subprocess
import sys
import tempfile
import unittest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SPEECH_STREAM = REPO_ROOT / "talk" / "tests" / "speech_stream.py"
RATE = 24000

# A stand-in player: records the WAV path it was handed, plays nothing.
RECORD_PLAYER = (
    f"{sys.executable} -c "
    '"import sys,pathlib;pathlib.Path(sys.argv[-1]+\'.played\').write_text(\'yes\')"'
)

CONFIG_TEMPLATE = """
[radio]
host = "test-invalid-host"
sample_rate = 24000

[station]
callsign = "W5TSU"
phonetic_words = ["whiskey", "five", "tango", "sierra", "uniform"]

[scripts]
id_text = "This is W5TSU."
signoff = "W5TSU clear."

[logging]
dir = "{log_dir}"
"""


@unittest.skipUnless(os.environ.get("TALK_MODELS_DIR"), "real-model test; set TALK_MODELS_DIR")
class TestReplyE2E(unittest.TestCase):
    def _run(self, phrase):
        tmp = tempfile.mkdtemp()
        log_dir = pathlib.Path(tmp) / "logs"
        cfg = pathlib.Path(tmp) / "config.toml"
        cfg.write_text(CONFIG_TEMPLATE.format(log_dir=log_dir))

        env = {k: v for k, v in os.environ.items() if k != "ANTHROPIC_API_KEY"}
        env["TALK_STREAM_CMD"] = (
            f"{sys.executable} {SPEECH_STREAM} {RATE} 1 {shlex.quote(phrase)}"
        )
        env["TALK_PLAYER_CMD"] = RECORD_PLAYER

        result = subprocess.run(
            [sys.executable, "-m", "talk", "--config", str(cfg)],
            cwd=REPO_ROOT, capture_output=True, text=True, timeout=240, env=env,
        )
        # Stream child exits 1 => "radio died" => loop reports and exits nonzero.
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        sessions = list(log_dir.iterdir())
        self.assertEqual(len(sessions), 1, result.stdout + result.stderr)
        records = [
            json.loads(l)
            for l in (sessions[0] / "session.jsonl").read_text().splitlines()
        ]
        return records, result

    def test_addressed_speech_produces_a_played_reply(self):
        records, result = self._run(
            "Whiskey five tango sierra uniform. Whiskey five tango sierra uniform. "
            "Radio check, are you there?"
        )

        utt = next(r for r in records if r["event"] == "utterance")
        self.assertTrue(utt["triggered"], utt)
        self.assertEqual(utt["trigger_kind"], "callsign")

        reply = next(r for r in records if r["event"] == "reply")
        self.assertFalse(reply["armed"])
        self.assertFalse(reply["refused"])
        self.assertTrue(pathlib.Path(reply["wav"]).exists())
        # The configured player was invoked with the reply WAV (rehearsal
        # plays locally; here the stand-in just records the path).
        self.assertTrue(pathlib.Path(reply["wav"] + ".played").exists())
        self.assertIn("rehearsal", result.stdout.lower())

    def test_unaddressed_speech_produces_no_reply(self):
        records, result = self._run(
            "Just some chatter about the weather on the band today, nothing to see here."
        )
        self.assertTrue(any(r["event"] == "utterance" for r in records))
        self.assertFalse(
            any(r["triggered"] for r in records if r["event"] == "utterance")
        )
        self.assertFalse(any(r["event"] == "reply" for r in records))


if __name__ == "__main__":
    unittest.main()
