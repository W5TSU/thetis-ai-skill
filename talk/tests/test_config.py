"""Config seam: talk.config.load(path) -> TalkConfig, or ConfigError."""

import pathlib
import re
import tempfile
import unittest

from talk import config

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]

MINIMAL = """
[radio]
host = "192.168.1.50"

[station]
callsign = "W5TSU"
phonetic_words = ["whiskey", "five", "tango", "sierra", "uniform"]

[scripts]
id_text = "This is W5TSU, automated voice assistant."
signoff = "W5TSU clear."
"""


def load_toml(text: str) -> config.TalkConfig:
    with tempfile.NamedTemporaryFile("w", suffix=".toml", delete=False) as f:
        f.write(text)
        path = f.name
    return config.load(path)


class TestLoad(unittest.TestCase):
    def test_minimal_config_gets_defaults(self):
        cfg = load_toml(MINIMAL)
        self.assertEqual(cfg.radio.host, "192.168.1.50")
        self.assertEqual(cfg.radio.cat_port, 13013)
        self.assertEqual(cfg.radio.tci_port, 50001)
        self.assertEqual(cfg.radio.rx, 0)
        self.assertEqual(cfg.radio.sample_rate, 24000)
        self.assertEqual(cfg.station.callsign, "W5TSU")
        self.assertEqual(cfg.station.wake_names, ())
        self.assertEqual(cfg.budgets.max_tx_seconds, 60)
        self.assertEqual(cfg.budgets.max_qso_seconds, 600)
        self.assertEqual(cfg.budgets.armed_session_seconds, 3600)
        self.assertEqual(cfg.claude.model, "claude-opus-5")
        self.assertTrue(cfg.logging.enabled)

    def test_example_config_parses(self):
        cfg = config.load(REPO_ROOT / "talk" / "config.toml.example")
        self.assertTrue(cfg.station.callsign)
        self.assertEqual(len(cfg.station.phonetic_words), 5)

    def test_missing_required_key_names_it(self):
        bad = MINIMAL.replace('callsign = "W5TSU"\n', "")
        with self.assertRaisesRegex(config.ConfigError, "callsign"):
            load_toml(bad)

    def test_unknown_key_rejected(self):
        with self.assertRaisesRegex(config.ConfigError, "frobnicate"):
            load_toml(MINIMAL + "\n[radio.frobnicate]\nx = 1\n")
        with self.assertRaisesRegex(config.ConfigError, "colour"):
            load_toml(MINIMAL + '\ncolour = "red"\n')

    def test_wrong_type_rejected(self):
        bad = MINIMAL.replace('host = "192.168.1.50"', "host = 5")
        with self.assertRaisesRegex(config.ConfigError, "host"):
            load_toml(bad)

    def test_bad_sample_rate_rejected(self):
        bad = MINIMAL.replace('host = "192.168.1.50"', 'host = "h"\nsample_rate = 44100')
        with self.assertRaisesRegex(config.ConfigError, "sample_rate"):
            load_toml(bad)

    def test_missing_file(self):
        with self.assertRaises(config.ConfigError):
            config.load("/nonexistent/talk.toml")


class TestConfirmPhraseDrift(unittest.TestCase):
    def test_matches_go_source_of_truth(self):
        go = (REPO_ROOT / "internal" / "safety" / "txgate.go").read_text()
        m = re.search(r'ConfirmPhrase\s*=\s*"([^"]+)"', go)
        self.assertIsNotNone(m, "ConfirmPhrase not found in txgate.go")
        from talk import constants

        self.assertEqual(constants.CONFIRM_PHRASE, m.group(1))


if __name__ == "__main__":
    unittest.main()
