"""Per-session station records: utterance/reply WAVs plus a JSONL transcript.

These double as the operator's FCC-facing record of what the station heard
and transmitted, so nothing here auto-prunes; deletion is a deliberate act.
"""

from __future__ import annotations

import json
import time
import wave
from array import array
from datetime import datetime, timezone
from pathlib import Path


class SessionLog:
    def __init__(self, base_dir: str | Path, enabled: bool = True):
        self.enabled = enabled
        self._counter = 0
        if not enabled:
            self.dir = None
            return
        stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
        self.dir = Path(base_dir) / stamp
        self.dir.mkdir(parents=True, exist_ok=True)
        self._jsonl = open(self.dir / "session.jsonl", "a")

    def write_wav(self, prefix: str, samples: array, sample_rate: int) -> Path | None:
        """Write float32 samples as 16-bit mono WAV; returns the path."""
        if not self.enabled:
            return None
        self._counter += 1
        path = self.dir / f"{self._counter:04d}-{prefix}.wav"
        pcm16 = array("h", (max(-32768, min(32767, int(s * 32767))) for s in samples))
        with wave.open(str(path), "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(2)
            w.setframerate(sample_rate)
            w.writeframes(pcm16.tobytes())
        return path

    def event(self, kind: str, **fields) -> None:
        if not self.enabled:
            return
        record = {"t": time.time(), "event": kind, **fields}
        self._jsonl.write(json.dumps(record) + "\n")
        self._jsonl.flush()

    def close(self) -> None:
        if self.enabled:
            self._jsonl.close()
