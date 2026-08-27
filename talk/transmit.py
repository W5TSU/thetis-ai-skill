"""Rehearsal playback and real transmission.

Player plays a reply WAV on the local speakers in rehearsal mode.
Transmitter runs `thetisctl tci tx-audio send` — that command keys PTT
itself and confirms its own unkey on completion, error, or SIGINT
(cmd/thetisctl/tci_cmd.go). abort() relies entirely on that: it sends
SIGINT and waits, and NEVER escalates to SIGKILL — killing the child would
skip its confirmed unkey and could leave the radio keyed. A child that
won't confirm within the grace period is left running and reported as
unconfirmed; the caller's job is to surface that loudly, not to kill it.
"""

from __future__ import annotations

import signal
import subprocess
from dataclasses import dataclass
from pathlib import Path

DEFAULT_PLAYER_CMD = ["aplay", "-q"]


class Player:
    def __init__(self, cmd: list[str] = DEFAULT_PLAYER_CMD):
        self._cmd = cmd

    def play(self, wav_path: str | Path) -> None:
        subprocess.run([*self._cmd, str(wav_path)], check=False)


@dataclass(frozen=True)
class TxResult:
    exit_code: int | None  # None means: still running, unkey unconfirmed
    saw_tx_off: bool


class Transmitter:
    def __init__(self, cmd_builder):
        """cmd_builder(wav_path) -> argv, e.g. building the thetisctl
        tci tx-audio send invocation for that reply WAV."""
        self._cmd_builder = cmd_builder
        self._child: subprocess.Popen | None = None

    def send(self, wav_path: str | Path) -> TxResult:
        self.send_async(wav_path)
        return self._wait_and_collect()

    def send_async(self, wav_path: str | Path) -> None:
        self._child = subprocess.Popen(
            self._cmd_builder(wav_path),
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )

    def is_done(self) -> bool:
        return self._child is None or self._child.poll() is not None

    def result(self) -> TxResult:
        """Collect the result of a child already confirmed done via is_done()."""
        return self._wait_and_collect()

    def abort(self, grace_seconds: float = 10.0) -> TxResult | None:
        """SIGINT the in-flight child and wait for its confirmed unkey.

        Sends a second SIGINT partway through the grace period in case the
        first was missed, but never sends anything stronger. Returns None
        if nothing was in flight.
        """
        if self._child is None or self._child.poll() is not None:
            self._child = None
            return None

        half = max(grace_seconds / 2, 0.05)
        self._child.send_signal(signal.SIGINT)
        try:
            self._child.wait(timeout=half)
        except subprocess.TimeoutExpired:
            self._child.send_signal(signal.SIGINT)
            try:
                self._child.wait(timeout=half)
            except subprocess.TimeoutExpired:
                return TxResult(exit_code=None, saw_tx_off=False)
        return self._wait_and_collect()

    def _wait_and_collect(self) -> TxResult:
        out, _ = self._child.communicate()
        code = self._child.returncode
        result = TxResult(exit_code=code, saw_tx_off="TX OFF" in out)
        self._child = None
        return result
