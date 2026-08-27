"""Rehearsal playback and (later slices) real transmission.

Player plays a reply WAV on the local speakers in rehearsal mode. The
Transmitter that keys the radio arrives in a later slice (armed transmit);
keeping it out of this module until then means no TX-capable code exists
before the arming ritual does.
"""

from __future__ import annotations

import subprocess
from pathlib import Path

DEFAULT_PLAYER_CMD = ["aplay", "-q"]


class Player:
    def __init__(self, cmd: list[str] = DEFAULT_PLAYER_CMD):
        self._cmd = cmd

    def play(self, wav_path: str | Path) -> None:
        subprocess.run([*self._cmd, str(wav_path)], check=False)
