"""The turn loop. This slice listens and recognizes: RX events -> VAD ->
transcribe -> wake match -> logged decision.

Later slices grow this into the full state machine (compose, synthesize,
transmit) with the QSO/ID/session clocks.
"""

from __future__ import annotations

import tempfile
from pathlib import Path

from talk.brain import QsoContext
from talk.config import StationConfig, TalkConfig
from talk.matcher import match as default_match
from talk.qsolog import SessionLog


class Session:
    def __init__(
        self,
        cfg: TalkConfig,
        stream,
        vad,
        log: SessionLog,
        sample_rate: int,
        out=print,
        transcriber=None,
        station: StationConfig | None = None,
        matcher=default_match,
        brain=None,
        synthesizer=None,
        player=None,
        transmitter=None,
        armed: bool = False,
    ):
        self._cfg = cfg
        self._stream = stream
        self._vad = vad
        self._log = log
        self._rate = sample_rate
        self._out = out
        self._transcriber = transcriber
        self._station = station or cfg.station
        self._matcher = matcher
        self._brain = brain
        self._synthesizer = synthesizer
        self._player = player
        self._transmitter = transmitter
        self._armed = armed
        self._qso = QsoContext()
        self._reply_seq = 0

    def run(self) -> int:
        """Consume the RX stream until it dies or the operator interrupts."""
        try:
            for ev in self._stream.events():
                if ev.kind == "audio":
                    utterance = self._vad.feed(ev.samples)
                    if utterance is not None:
                        self._on_utterance(utterance)
                elif ev.kind == "restart":
                    self._flush_pending_speech()
                    self._log.event("stream-gap")
                    self._out("rx stream hit planned expiry; restarted (gap logged)")
                elif ev.kind == "stalled":
                    self._flush_pending_speech()
                    self._log.event("rx-stalled")
                    self._out("WARNING: rx stream stalled (no data from radio); restarting")
                elif ev.kind == "dead":
                    self._flush_pending_speech()
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

    def _flush_pending_speech(self) -> None:
        u = self._vad.flush()
        if u is not None:
            self._on_utterance(u)

    def _on_utterance(self, u) -> None:
        duration = len(u.samples) / self._rate
        wav = self._log.write_wav("rx-utterance", u.samples, self._rate)
        fields = {
            "start": round(u.start, 3),
            "duration": round(duration, 3),
            "forced_cut": u.forced_cut,
            "wav": str(wav) if wav else None,
        }
        cut = " (force-cut)" if u.forced_cut else ""
        line = f"[{u.start:8.1f}s] utterance {duration:4.1f}s{cut}"

        if self._transcriber is not None:
            transcript = self._transcriber.transcribe(u.samples, self._rate)
            decision = self._matcher(transcript.text, self._station)
            fields.update(
                transcript=transcript.text,
                triggered=decision.triggered,
                trigger_kind=decision.kind,
                score=round(decision.score, 3),
            )
            status = f"triggered ({decision.kind})" if decision.triggered else "silent"
            line += f' — "{transcript.text}" [{status}]'

        self._log.event("utterance", **fields)
        self._out(line)

        if self._transcriber is not None and decision.triggered:
            self._reply(transcript.text)

    def _reply(self, heard_text: str) -> None:
        if self._brain is None:
            return
        decision = self._brain.compose(heard_text, self._qso)
        intent, source, reply_text = decision.intent, decision.source, decision.reply_text

        if self._synthesizer is None:
            self._log.event(
                "reply", intent=intent, intent_source=source, text=reply_text,
                armed=self._armed, synthesized=False,
            )
            self._out(f'  -> reply ({source}): "{reply_text}" [not synthesized]')
            return

        self._reply_seq += 1
        wav_dir = self._log.dir if self._log.enabled else Path(tempfile.gettempdir())
        out_path = wav_dir / f"{self._reply_seq:04d}-reply.wav"
        duration = self._synthesizer.synthesize(reply_text, out_path)
        self._log.event(
            "reply",
            intent=intent,
            intent_source=source,
            text=reply_text,
            duration=round(duration, 3),
            wav=str(out_path),
            armed=self._armed,
        )

        if self._armed:
            assert self._transmitter is not None, "armed session requires a transmitter"
            self._transmitter.send(out_path)
            self._out(f'  -> TX ({source}, {duration:.1f}s): "{reply_text}"')
        elif self._player is not None:
            self._player.play(out_path)
            self._out(f'  -> reply ({source}, {duration:.1f}s, rehearsal): "{reply_text}"')
