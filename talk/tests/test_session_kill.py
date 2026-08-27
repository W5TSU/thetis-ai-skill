"""Session seam: a kill switch triggered mid-transmission aborts the in-
flight TX (confirmed unkey) and disarms, rather than waiting for it to
finish naturally; triggered while just listening stops the session.
"""

import json
import pathlib
import tempfile
import time
import unittest
from array import array

from talk.audio_rx import RxEvent
from talk.brain import Decision
from talk.config import RadioConfig, ScriptsConfig, StationConfig, TalkConfig, VADConfig
from talk.qsolog import SessionLog
from talk.safety import Clocks
from talk.session import Session
from talk.transcribe import Transcript
from talk.transmit import TxResult
from talk.vad import EnergyVAD

RATE = 24000
STATION = StationConfig(
    callsign="W5TSU",
    phonetic_words=("whiskey", "five", "tango", "sierra", "uniform"),
)
SCRIPTS = ScriptsConfig(id_text="id", signoff="clear", fallback_reply="say again")


def tone(seconds, amplitude):
    n = int(seconds * RATE)
    return array("f", [amplitude if i % 2 else -amplitude for i in range(n)])


class ScriptedStream:
    def __init__(self, events):
        self._events = events
        self.stopped = False

    def events(self):
        return iter(self._events)

    def stop(self):
        self.stopped = True


class ScriptedTranscriber:
    def __init__(self, texts):
        self._texts = list(texts)

    def transcribe(self, samples, sample_rate):
        return Transcript(text=self._texts.pop(0), no_speech_prob=0.0, avg_confidence=-0.1)


class FakeBrain:
    def compose(self, heard, qso):
        qso.add_turn(heard, "a reply")
        return Decision("a reply", "claude", None)


class FakeSynthesizer:
    def synthesize(self, text, out_path):
        pathlib.Path(out_path).write_bytes(b"FAKE")
        return 5.0


class AlwaysTriggeredKillSwitch:
    """Simulates a keypress that arrives immediately."""

    def triggered(self):
        return True


class KillOnceTransmitStarts:
    """Simulates a keypress that lands only after transmission has begun,
    so the listening phase runs normally and the abort path is exercised."""

    def __init__(self, transmitter):
        self._transmitter = transmitter

    def triggered(self):
        return self._transmitter.sent_async


class NeverTriggeredKillSwitch:
    def triggered(self):
        return False


class StallWatchingStream(ScriptedStream):
    """Records every set_stall_suspended call, standing in for RxStream's
    real stall watchdog toggle (audio_rx.RxStream.set_stall_suspended)."""

    def __init__(self, events):
        super().__init__(events)
        self.suspend_calls: list[bool] = []

    def set_stall_suspended(self, suspended: bool) -> None:
        self.suspend_calls.append(suspended)


class ImmediateTransmitter:
    """A transmitter that "completes" the instant it's checked."""

    def __init__(self):
        self.sent_async = False

    def send_async(self, wav_path):
        self.sent_async = True

    def is_done(self):
        return True

    def result(self):
        return TxResult(exit_code=0, saw_tx_off=True)


class SlowTransmitter:
    """Never finishes on its own; only abort() ends it."""

    def __init__(self):
        self.aborted = False
        self.sent_async = False

    def send_async(self, wav_path):
        self.sent_async = True

    def is_done(self):
        return False

    def abort(self, grace_seconds=10.0):
        self.aborted = True
        return TxResult(exit_code=130, saw_tx_off=True)


def cfg():
    return TalkConfig(radio=RadioConfig(host="x"), station=STATION, scripts=SCRIPTS)


class TestKillDuringTransmit(unittest.TestCase):
    def test_kill_switch_aborts_in_flight_transmission(self):
        events = [
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("audio", tone(0.6, 0.2)),
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("dead"),
        ]
        transmitter = SlowTransmitter()
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(),
                stream=ScriptedStream(events),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=FakeBrain(),
                synthesizer=FakeSynthesizer(),
                transmitter=transmitter,
                armed=True,
                clocks=Clocks(cfg().budgets),
                kill_switch=KillOnceTransmitStarts(transmitter),
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]

        self.assertTrue(transmitter.sent_async)
        self.assertTrue(transmitter.aborted)
        self.assertTrue(any(r["event"] == "kill-triggered" for r in records))


class TestStallWatchdogSuspendedDuringTx(unittest.TestCase):
    """RxStream keeps streaming (near-silent) audio while we transmit, so
    without suspending its stall watchdog around a TX, a transmission
    longer than the stall timeout would spuriously look like a dead radio
    link. See audio_rx.RxStream's own docstring on this."""

    def test_suspended_around_a_transmission_and_resumed_after(self):
        events = [
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("audio", tone(0.6, 0.2)),
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("dead"),
        ]
        stream = StallWatchingStream(events)
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(),
                stream=stream,
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=FakeBrain(),
                synthesizer=FakeSynthesizer(),
                transmitter=ImmediateTransmitter(),
                armed=True,
                clocks=Clocks(cfg().budgets),
            )
            session.run()
        self.assertEqual(stream.suspend_calls, [True, False])

    def test_suspended_and_resumed_even_when_the_kill_switch_aborts_it(self):
        events = [
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("audio", tone(0.6, 0.2)),
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("dead"),
        ]
        stream = StallWatchingStream(events)
        transmitter = SlowTransmitter()
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(),
                stream=stream,
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=FakeBrain(),
                synthesizer=FakeSynthesizer(),
                transmitter=transmitter,
                armed=True,
                clocks=Clocks(cfg().budgets),
                kill_switch=KillOnceTransmitStarts(transmitter),
            )
            session.run()
        self.assertEqual(stream.suspend_calls, [True, False])


class TestKillWhileListening(unittest.TestCase):
    def test_kill_switch_stops_the_session(self):
        events = [RxEvent("audio", tone(1.0, 0.01)) for _ in range(5)] + [RxEvent("dead")]
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            stream = ScriptedStream(events)
            session = Session(
                cfg(),
                stream=stream,
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                clocks=Clocks(cfg().budgets),
                kill_switch=AlwaysTriggeredKillSwitch(),
            )
            code = session.run()
        self.assertEqual(code, 130)
        self.assertTrue(stream.stopped)

    def test_no_kill_switch_runs_to_completion_normally(self):
        events = [RxEvent("audio", tone(1.0, 0.01)) for _ in range(3)] + [RxEvent("dead")]
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(),
                stream=ScriptedStream(events),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                clocks=Clocks(cfg().budgets),
                kill_switch=NeverTriggeredKillSwitch(),
            )
            code = session.run()
        self.assertEqual(code, 1)  # ends via the scripted "dead" event, not the kill switch


if __name__ == "__main__":
    unittest.main()
