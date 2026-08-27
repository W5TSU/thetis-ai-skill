"""The turn loop. This slice listens and recognizes: RX events -> VAD ->
transcribe -> wake match -> logged decision.

Later slices grow this into the full state machine (compose, synthesize,
transmit) with the QSO/ID/session clocks.
"""

from __future__ import annotations

import tempfile
import time
from pathlib import Path

from talk.brain import QsoContext
from talk.config import StationConfig, TalkConfig
from talk.matcher import match as default_match
from talk.qsolog import SessionLog
from talk.safety import Clocks
from talk.tts import synthesize_capped


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
        clocks: Clocks | None = None,
        pretx_check=None,
        kill_switch=None,
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
        self._clocks = clocks or Clocks(cfg.budgets)
        self._pretx_check = pretx_check
        self._kill_switch = kill_switch

    def run(self) -> int:
        """Consume the RX stream until it dies or the operator interrupts."""
        try:
            for ev in self._stream.events():
                if self._kill_switch is not None and self._kill_switch.triggered():
                    self._out("KILL: stopping (keypress/interrupt)")
                    self._log.event("kill-triggered")
                    self._stream.stop()
                    return 130
                if ev.kind == "audio":
                    utterance = self._vad.feed(ev.samples)
                    if utterance is not None:
                        self._on_utterance(utterance)
                    elif self._clocks.qso_idle_expired():
                        self._clocks.close_qso()
                        self._qso = QsoContext()
                        self._log.event("qso-idle-close")
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
            self._clocks.note_triggered_turn()
            self._reply(transcript.text)

    def _compose_reply(self, heard_text: str) -> tuple[str, str, str | None]:
        """Returns (text, source, intent). source is one of rule/claude/
        fallback/signoff/session-expired; the last two override the brain."""
        if self._armed and self._clocks.session_expired():
            return self._cfg.scripts.signoff, "session-expired", None
        if self._clocks.qso_over_budget():
            return self._cfg.scripts.signoff, "signoff", None
        if self._brain is None:
            return self._cfg.scripts.fallback_reply, "fallback", None
        decision = self._brain.compose(heard_text, self._qso)
        return decision.reply_text, decision.source, decision.intent

    def _reply(self, heard_text: str) -> None:
        reply_text, source, intent = self._compose_reply(heard_text)
        is_signoff = source in ("signoff", "session-expired")

        if self._clocks.needs_id() or is_signoff:
            # ID goes first: sentence-drop shortening below trims from the
            # end, and the ID must never be the part that gets dropped.
            reply_text = f"{self._cfg.scripts.id_text} {reply_text}"

        if self._synthesizer is None:
            self._log.event(
                "reply", intent=intent, intent_source=source, text=reply_text,
                armed=self._armed, synthesized=False,
            )
            self._out(f'  -> reply ({source}): "{reply_text}" [not synthesized]')
            if is_signoff:
                self._close_qso()
            return

        self._reply_seq += 1
        wav_dir = self._log.dir if self._log.enabled else Path(tempfile.gettempdir())
        out_path = wav_dir / f"{self._reply_seq:04d}-reply.wav"
        duration, refused, reply_text = synthesize_capped(
            self._synthesizer, reply_text, out_path, self._cfg.budgets.max_tx_seconds
        )
        self._log.event(
            "reply",
            intent=intent,
            intent_source=source,
            text=reply_text,
            duration=round(duration, 3),
            wav=str(out_path),
            armed=self._armed,
            refused=refused,
        )
        if refused:
            self._out(f'  -> REFUSED (too long even after shortening): "{reply_text}"')
            if is_signoff:
                self._close_qso()
            return

        sent = self._send_or_play(out_path, reply_text, source, duration)
        if sent:
            self._clocks.mark_id_sent()
        if is_signoff:
            self._close_qso()
        if source == "session-expired" and sent:
            self._armed = False
            self._log.event("disarm", reason="session-expired")
            self._out("session expired; disarmed after the grace sign-off")

    def _send_or_play(self, out_path: Path, reply_text: str, source: str, duration: float) -> bool:
        """Runs the pre-TX check (armed or not, for observability), then
        transmits (armed) or plays locally (rehearsal). Returns whether a
        transmission/playback actually happened."""
        if self._pretx_check is not None:
            result = self._pretx_check.check(armed=self._armed)
            if result.reason is not None:
                self._log.event("pretx-check", ok=result.ok, reason=result.reason)
            if not result.ok:
                if self._armed:
                    self._armed = False
                    self._log.event("disarm", reason=result.reason)
                    self._out(f"WARNING: pre-TX check failed ({result.reason}); disarmed, not transmitting")
                else:
                    self._out(f"WARNING: pre-TX check failed ({result.reason}) [rehearsal, no action]")
                return False

        if self._armed:
            assert self._transmitter is not None, "armed session requires a transmitter"
            self._out(f'  -> TX ({source}, {duration:.1f}s): "{reply_text}"')
            result = self._transmit_with_kill_watch(out_path)
            if result.exit_code != 0 or not result.saw_tx_off:
                self._armed = False
                self._log.event(
                    "disarm", reason="tx-anomaly",
                    exit_code=result.exit_code, saw_tx_off=result.saw_tx_off,
                )
                self._out(
                    f"WARNING: TX did not confirm a clean unkey (exit={result.exit_code}); "
                    "disarmed — CHECK THE RADIO NOW"
                )
            return True
        elif self._player is not None:
            self._player.play(out_path)
            self._out(f'  -> reply ({source}, {duration:.1f}s, rehearsal): "{reply_text}"')
            return True
        return False

    def _transmit_with_kill_watch(self, out_path: Path):
        """Sends without blocking so a kill switch can abort mid-transmission
        instead of only being checked once the child finishes on its own."""
        self._transmitter.send_async(out_path)
        while True:
            if self._transmitter.is_done():
                return self._transmitter.result()
            if self._kill_switch is not None and self._kill_switch.triggered():
                self._out("KILL: aborting in-flight transmission (keypress/interrupt)")
                self._log.event("kill-triggered")
                result = self._transmitter.abort(grace_seconds=10)
                self._armed = False
                return result
            time.sleep(0.05)

    def _close_qso(self) -> None:
        self._clocks.close_qso()
        self._qso = QsoContext()
