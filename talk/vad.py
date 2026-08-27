"""Energy-based voice-activity detection over the RX PCM stream.

Stdlib-only on purpose: the offline test suite must run without numpy or the
speech wheels. Block-wise RMS against an adaptive noise floor; the floor is an
exponential moving average updated only outside speech, so SSB band noise sets
the threshold and voice peaks ride above it.
"""

from __future__ import annotations

import math
from array import array
from dataclasses import dataclass

from talk.config import VADConfig

BLOCK_MS = 20
# Below this RMS the input is considered dead air regardless of the adaptive
# floor (prevents a near-zero floor from turning noise into "speech").
ABSOLUTE_FLOOR = 0.003
FLOOR_ALPHA = 0.05


@dataclass(frozen=True)
class Utterance:
    samples: array  # float32, at the stream sample rate
    start: float  # seconds since stream start
    end: float
    forced_cut: bool


class EnergyVAD:
    def __init__(self, cfg: VADConfig, sample_rate: int):
        self._cfg = cfg
        self._rate = sample_rate
        self._block = max(1, sample_rate * BLOCK_MS // 1000)
        self._pending = array("f")
        self._blocks_seen = 0

        self._floor: float | None = None
        self._in_speech = False
        self._onset_run = 0
        self._hang_run = 0
        self._preroll: list[array] = []
        self._preroll_blocks = max(1, cfg.preroll_ms // BLOCK_MS)
        self._onset_blocks = max(1, cfg.onset_ms // BLOCK_MS)
        self._hangover_blocks = max(1, cfg.hangover_ms // BLOCK_MS)
        self._speech: array = array("f")
        self._speech_start_block = 0
        self._loud_blocks = 0

    def feed(self, samples) -> Utterance | None:
        """Feed any amount of PCM; returns a completed Utterance when one ends."""
        self._pending.extend(samples)
        result = None
        while len(self._pending) >= self._block and result is None:
            block = self._pending[: self._block]
            del self._pending[: self._block]
            result = self._feed_block(block)
        return result

    def _feed_block(self, block: array) -> Utterance | None:
        rms = math.sqrt(sum(s * s for s in block) / len(block))
        self._blocks_seen += 1

        if self._floor is None:
            self._floor = rms
        threshold = max(ABSOLUTE_FLOOR, self._floor * self._cfg.threshold_ratio)
        loud = rms > threshold

        if not self._in_speech:
            if not loud:
                # Only quiet blocks move the floor. Adapting on loud blocks
                # too would let a rising threshold chase a rising speech
                # envelope and starve onset of its 5 consecutive hits.
                self._floor = (1 - FLOOR_ALPHA) * self._floor + FLOOR_ALPHA * rms
            self._preroll.append(block)
            if len(self._preroll) > self._preroll_blocks + self._onset_blocks:
                self._preroll.pop(0)
            if loud:
                self._onset_run += 1
                if self._onset_run >= self._onset_blocks:
                    self._start_speech()
            else:
                self._onset_run = 0
            return None

        # In speech: accumulate, watch for the end or the force-cut.
        self._speech.extend(block)
        if loud:
            self._hang_run = 0
            self._loud_blocks += 1
        else:
            self._hang_run += 1

        if len(self._speech) >= self._cfg.max_utterance_seconds * self._rate:
            return self._end_speech(forced=True)
        if self._hang_run >= self._hangover_blocks:
            return self._end_speech(forced=False)
        return None

    def _start_speech(self) -> None:
        self._in_speech = True
        self._hang_run = 0
        self._onset_run = 0
        self._speech = array("f")
        for b in self._preroll:
            self._speech.extend(b)
        self._speech_start_block = self._blocks_seen - len(self._preroll)
        self._loud_blocks = self._onset_blocks
        self._preroll = []

    def flush(self) -> Utterance | None:
        """Return any in-progress speech as a force-cut Utterance.

        Call this when the stream ends or restarts: real audio, unlike the
        synthetic fixtures above, doesn't reliably trail off into hangover-
        length silence, so without a flush a stream cut mid-utterance would
        silently lose whatever was said.
        """
        if not self._in_speech:
            return None
        return self._end_speech(forced=True)

    def _end_speech(self, forced: bool) -> Utterance | None:
        samples = self._speech
        start_block = self._speech_start_block
        self._in_speech = False
        self._speech = array("f")
        self._hang_run = 0

        speech_ms = self._loud_blocks * BLOCK_MS
        if not forced and speech_ms < self._cfg.min_utterance_ms:
            return None
        start = start_block * self._block / self._rate
        return Utterance(
            samples=samples,
            start=start,
            end=start + len(samples) / self._rate,
            forced_cut=forced,
        )
