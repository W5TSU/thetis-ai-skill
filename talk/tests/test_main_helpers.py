"""Pure helpers in talk/__main__.py — no subprocess, no models, no radio."""

import os
import unittest
from unittest import mock

from talk.__main__ import player_from_env
from talk.transmit import Player


class TestPlayerFromEnv(unittest.TestCase):
    def test_defaults_to_the_builtin_player_when_unset(self):
        env = {k: v for k, v in os.environ.items() if k != "TALK_PLAYER_CMD"}
        with mock.patch.dict(os.environ, env, clear=True):
            self.assertEqual(player_from_env()._cmd, Player()._cmd)

    def test_uses_TALK_PLAYER_CMD_split_into_argv(self):
        with mock.patch.dict(os.environ, {"TALK_PLAYER_CMD": "paplay --raw -"}):
            self.assertEqual(player_from_env()._cmd, ["paplay", "--raw", "-"])


if __name__ == "__main__":
    unittest.main()
