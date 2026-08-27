"""Manage the long-running RX audio stream child and classify its lifecycle.

The child (normally `thetisctl tci rx-audio stream ... --duration 4h`) writes
raw float32 LE PCM to stdout and nothing else. Events tell the caller what the
radio link is doing:

  audio    - decoded samples
  restart  - the child hit its planned --duration expiry (exit 0) and was
             respawned; audio between the EOF and respawn is lost
  stalled  - the child is alive but produced no bytes for stall_timeout;
             the child is killed and respawned (a stalled TCI link is
             indistinguishable from a dead radio, so armed callers disarm)
  dead     - the child exited nonzero or could not be spawned; the stream ends

Silence is NOT absence of data — Thetis streams frames continuously and
silence arrives as near-zero samples. The stall watchdog can be suspended
while our own TX mutes RX legitimately.
"""

from __future__ import annotations

import queue
import subprocess
import threading
import time
from array import array
from dataclasses import dataclass
from typing import Iterator

_EOF = object()


@dataclass(frozen=True)
class RxEvent:
    kind: str  # "audio" | "restart" | "stalled" | "dead"
    samples: array | None = None


class RxStream:
    def __init__(self, cmd: list[str], stall_timeout: float = 2.5):
        self._cmd = cmd
        self._stall_timeout = stall_timeout
        self._stall_suspended = False
        self._stopped = False
        self._child: subprocess.Popen | None = None

    def set_stall_suspended(self, suspended: bool) -> None:
        self._stall_suspended = suspended
        if not suspended:
            self._last_data = time.monotonic()

    def stop(self) -> None:
        self._stopped = True
        child = self._child
        if child is not None and child.poll() is None:
            child.terminate()

    def events(self) -> Iterator[RxEvent]:
        while not self._stopped:
            try:
                self._child = subprocess.Popen(
                    self._cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.DEVNULL,
                )
            except OSError:
                yield RxEvent("dead")
                return

            q: queue.Queue = queue.Queue()
            reader = threading.Thread(
                target=self._read, args=(self._child, q), daemon=True
            )
            reader.start()
            self._last_data = time.monotonic()

            ended = None  # set to the final event kind for this child
            leftover = b""
            while ended is None:
                if self._stopped:
                    self.stop()
                try:
                    item = q.get(timeout=0.1)
                except queue.Empty:
                    if self._stopped:
                        return
                    if (
                        not self._stall_suspended
                        and time.monotonic() - self._last_data > self._stall_timeout
                    ):
                        self._child.kill()
                        self._child.wait()
                        ended = "stalled"
                    continue
                if item is _EOF:
                    code = self._child.wait()
                    if self._stopped:
                        return
                    ended = "restart" if code == 0 else "dead"
                    continue
                self._last_data = time.monotonic()
                data = leftover + item
                usable = len(data) - (len(data) % 4)
                leftover = data[usable:]
                if usable:
                    samples = array("f")
                    samples.frombytes(data[:usable])
                    yield RxEvent("audio", samples)

            if ended == "dead":
                yield RxEvent("dead")
                return
            yield RxEvent(ended)  # "restart" or "stalled"; loop respawns

    @staticmethod
    def _read(child: subprocess.Popen, q: queue.Queue) -> None:
        while True:
            data = child.stdout.read(4096)
            if not data:
                q.put(_EOF)
                return
            q.put(data)
