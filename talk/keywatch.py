"""The armed-session kill switch: any keypress or SIGINT stops transmitting.

Armed mode requires a real TTY — a kill switch that can't read keys is no
kill switch, so construction refuses a non-interactive stream outright
unless the caller explicitly opts out (tests, and nothing else).
"""

from __future__ import annotations

import select
import signal
import sys
import termios
import tty


class KillSwitch:
    def __init__(self, stream=None, tty_required: bool = True):
        self._stream = stream if stream is not None else sys.stdin
        self._tty_required = tty_required
        self._is_tty = hasattr(self._stream, "isatty") and self._stream.isatty()
        if tty_required and not self._is_tty:
            raise RuntimeError(
                "armed mode requires a real terminal (stdin is not a TTY) — "
                "the kill switch could not read keypresses"
            )
        self._flag = False
        self._saved_termios = None
        self._prev_sigint = None

    def __enter__(self) -> "KillSwitch":
        self._prev_sigint = signal.signal(signal.SIGINT, self._on_sigint)
        if self._is_tty:
            fd = self._stream.fileno()
            self._saved_termios = termios.tcgetattr(fd)
            tty.setcbreak(fd)
        return self

    def __exit__(self, *exc) -> None:
        if self._saved_termios is not None:
            termios.tcsetattr(self._stream.fileno(), termios.TCSADRAIN, self._saved_termios)
        signal.signal(signal.SIGINT, self._prev_sigint)

    def _on_sigint(self, signum, frame) -> None:
        self._flag = True

    def triggered(self) -> bool:
        if self._flag:
            return True
        readable, _, _ = select.select([self._stream], [], [], 0)
        if readable:
            try:
                self._stream.read(1)
            except OSError:
                pass
            self._flag = True
        return self._flag
