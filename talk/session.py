"""The turn loop. This slice listens: RX events -> VAD -> logged Utterances.

Later slices grow this into the full state machine (transcribe, match,
compose, synthesize, transmit) with the QSO/ID/session clocks.
"""

from __future__ import annotations

from talk.config import TalkConfig
from talk.qsolog import SessionLog


class Session:
    def __init__(self, cfg: TalkConfig, stream, vad, log: SessionLog, sample_rate: int, out=print):
        self._cfg = cfg
        self._stream = stream
        self._vad = vad
        self._log = log
        self._rate = sample_rate
        self._out = out

    def run(self) -> int:
        """Consume the RX stream until it dies or the operator interrupts."""
        try:
            for ev in self._stream.events():
                if ev.kind == "audio":
                    utterance = self._vad.feed(ev.samples)
                    if utterance is not None:
                        self._on_utterance(utterance)
                elif ev.kind == "restart":
                    self._log.event("stream-gap")
                    self._out("rx stream hit planned expiry; restarted (gap logged)")
                elif ev.kind == "stalled":
                    self._log.event("rx-stalled")
                    self._out("WARNING: rx stream stalled (no data from radio); restarting")
                elif ev.kind == "dead":
                    self._log.event("rx-dead")
                    self._out("ERROR: rx stream died; radio unreachable")
                    return 1
            return 0
        except KeyboardInterrupt:
            self._out("interrupted; stopping")
            self._stream.stop()
            return 130
        finally:
            self._log.close()

    def _on_utterance(self, u) -> None:
        duration = len(u.samples) / self._rate
        wav = self._log.write_wav("rx-utterance", u.samples, self._rate)
        self._log.event(
            "utterance",
            start=round(u.start, 3),
            duration=round(duration, 3),
            forced_cut=u.forced_cut,
            wav=str(wav) if wav else None,
        )
        cut = " (force-cut)" if u.forced_cut else ""
        self._out(f"[{u.start:8.1f}s] utterance {duration:4.1f}s{cut}")
