"""CLI seam: python -m talk --config <file> [--check]."""

import pathlib
import subprocess
import sys
import tempfile
import unittest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
EXAMPLE = REPO_ROOT / "talk" / "config.toml.example"


def run_cli(*args):
    return subprocess.run(
        [sys.executable, "-m", "talk", *args],
        cwd=REPO_ROOT,
        capture_output=True,
        text=True,
        timeout=30,
    )


class TestCLI(unittest.TestCase):
    def test_check_prints_banner_and_exits_zero(self):
        result = run_cli("--config", str(EXAMPLE), "--check")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("W5TSU", result.stdout)
        self.assertIn("REHEARSAL", result.stdout)

    def test_bad_config_is_a_clear_error_not_a_traceback(self):
        with tempfile.NamedTemporaryFile("w", suffix=".toml", delete=False) as f:
            f.write(
                '[radio]\nhost = 5\n'
                '[station]\ncallsign = "W5TSU"\n'
                'phonetic_words = ["whiskey", "five"]\n'
                '[scripts]\nid_text = "id"\nsignoff = "clear"\n'
            )
            bad = f.name
        result = run_cli("--config", bad, "--check")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("host", result.stderr)
        self.assertNotIn("Traceback", result.stderr)

    def test_missing_config_flag_fails(self):
        result = run_cli("--check")
        self.assertNotEqual(result.returncode, 0)

    def test_armed_without_confirm_tx_refused(self):
        result = run_cli("--config", str(EXAMPLE), "--armed")
        self.assertEqual(result.returncode, 2)
        self.assertIn("confirm-tx", result.stderr)

    def test_armed_with_wrong_confirm_tx_refused(self):
        result = run_cli("--config", str(EXAMPLE), "--armed", "--confirm-tx", "nope")
        self.assertEqual(result.returncode, 2)

    def test_armed_refuses_non_tty_before_any_radio_io(self):
        result = subprocess.run(
            [sys.executable, "-m", "talk", "--config", str(EXAMPLE),
             "--armed", "--confirm-tx", "I-UNDERSTAND-THIS-KEYS-THE-RADIO"],
            cwd=REPO_ROOT, capture_output=True, text=True, timeout=30, stdin=subprocess.DEVNULL,
        )
        self.assertEqual(result.returncode, 2)
        self.assertIn("TTY", result.stderr)
        # Refused before printing the arm banner or touching the network.
        self.assertNotIn("ARMED", result.stdout)


if __name__ == "__main__":
    unittest.main()
