"""Player seam: rehearsal playback via a subprocess player command."""

import sys
import tempfile
import unittest
from pathlib import Path

from talk.transmit import Player

RECORD_ARGS = (
    "import sys, pathlib;"
    "pathlib.Path(sys.argv[-1] + '.played').write_text('yes')"
)


class TestPlayer(unittest.TestCase):
    def test_play_invokes_configured_command_with_the_wav_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            wav = Path(tmp) / "reply.wav"
            wav.write_bytes(b"RIFF....WAVEfmt ")
            player = Player(cmd=[sys.executable, "-c", RECORD_ARGS])
            player.play(wav)
            self.assertTrue((Path(str(wav) + ".played")).exists())

    def test_missing_player_binary_raises_clear_error(self):
        player = Player(cmd=["/nonexistent/definitely-not-a-binary"])
        with self.assertRaises(FileNotFoundError):
            player.play(Path("/tmp/whatever.wav"))


if __name__ == "__main__":
    unittest.main()
