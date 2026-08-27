"""Session seam, safety core: QSO cap + sign-off, ID timing, session expiry
grace, 60s refusal, and disarm on a failed pre-TX check. All driven by a
fake clock so timing is deterministic and the whole suite runs instantly.
"""

import json
import pathlib
import tempfile
import unittest
from array import array
from dataclasses import dataclass

from talk.audio_rx import RxEvent
from talk.brain import Decision
from talk.config import BudgetsConfig, RadioConfig, ScriptsConfig, StationConfig, TalkConfig, VADConfig
from talk.qsolog import SessionLog
from talk.safety import Clocks, PreTxCheck, RadioStatus
from talk.session import Session
from talk.transcribe import Transcript
from talk.transmit import TxResult
from talk.vad import EnergyVAD

RATE = 24000
STATION = StationConfig(
    callsign="W5TSU",
    phonetic_words=("whiskey", "five", "tango", "sierra", "uniform"),
    wake_names=("thetis",),
)
SCRIPTS = ScriptsConfig(id_text="This is W5TSU.", signoff="W5TSU clear.", fallback_reply="Say again.")


def tone(seconds, amplitude):
    n = int(seconds * RATE)
    return array("f", [amplitude if i % 2 else -amplitude for i in range(n)])


def one_utterance_events():
    return [
        RxEvent("audio", tone(1.0, 0.01)),
        RxEvent("audio", tone(0.6, 0.2)),
        RxEvent("audio", tone(1.0, 0.01)),
    ]


class ScriptedStream:
    def __init__(self, events):
        self._events = events

    def events(self):
        return iter(self._events)

    def stop(self):
        pass


class ScriptedTranscriber:
    def __init__(self, texts):
        self._texts = list(texts)

    def transcribe(self, samples, sample_rate):
        return Transcript(text=self._texts.pop(0), no_speech_prob=0.0, avg_confidence=-0.1)


class FakeBrain:
    def __init__(self, text="a reply"):
        self._text = text

    def compose(self, heard, qso):
        qso.add_turn(heard, self._text)
        return Decision(self._text, "claude", None)


class FakeSynthesizer:
    def __init__(self, seconds_per_word=0.3):
        self.calls = []
        self._spw = seconds_per_word

    def synthesize(self, text, out_path):
        self.calls.append(text)
        pathlib.Path(out_path).write_bytes(b"FAKE")
        return len(text.split()) * self._spw


class FakePlayer:
    def __init__(self):
        self.played = []

    def play(self, wav_path):
        self.played.append(str(wav_path))


class FakeTransmitter:
    def __init__(self):
        self.sent = []

    def send_async(self, wav_path):
        self.sent.append(str(wav_path))

    def is_done(self):
        return True

    def result(self):
        return TxResult(exit_code=0, saw_tx_off=True)


class FakeClock:
    def __init__(self, start=0.0):
        self.t = start

    def __call__(self):
        return self.t

    def advance(self, seconds):
        self.t += seconds


def cfg(budgets=None):
    return TalkConfig(
        radio=RadioConfig(host="x"), station=STATION, scripts=SCRIPTS,
        budgets=budgets or BudgetsConfig(),
    )


def run_session(events, texts, budgets=None, clock=None, armed=False, pretx_check=None, transmitter=None):
    with tempfile.TemporaryDirectory() as tmp:
        log = SessionLog(tmp, enabled=True)
        events = list(events) + [RxEvent("dead")]
        session = Session(
            cfg(budgets),
            stream=ScriptedStream(events),
            vad=EnergyVAD(VADConfig(), RATE),
            log=log,
            sample_rate=RATE,
            out=lambda *_: None,
            transcriber=ScriptedTranscriber(texts),
            station=STATION,
            brain=FakeBrain(),
            synthesizer=FakeSynthesizer(),
            player=FakePlayer(),
            transmitter=transmitter or FakeTransmitter(),
            armed=armed,
            clocks=Clocks(budgets or BudgetsConfig(), clock=clock or FakeClock()),
            pretx_check=pretx_check,
        )
        session.run()
        records = []
        for p in pathlib.Path(tmp).iterdir():
            records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
        return records


class TestQsoCap(unittest.TestCase):
    def test_qso_over_cap_gets_signoff_instead_of_composed_reply(self):
        budgets = BudgetsConfig(max_qso_seconds=10)
        clock = FakeClock()

        # Two triggered turns; the clock jumps past the cap between them.
        class SteppedStream:
            def events(self):
                first_batch = list(one_utterance_events())
                for e in first_batch:
                    yield e
                clock.advance(11)
                second_batch = list(one_utterance_events())
                for e in second_batch:
                    yield e
                yield RxEvent("dead")

            def stop(self):
                pass

        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(budgets),
                stream=SteppedStream(),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(
                    ["whiskey five tango sierra uniform first", "whiskey five tango sierra uniform second"]
                ),
                station=STATION,
                brain=FakeBrain("a normal reply"),
                synthesizer=FakeSynthesizer(),
                player=FakePlayer(),
                clocks=Clocks(budgets, clock=clock),
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
        replies = [r for r in records if r["event"] == "reply"]
        self.assertEqual(len(replies), 2)
        self.assertEqual(replies[0]["intent_source"], "claude")
        self.assertEqual(replies[1]["intent_source"], "signoff")
        self.assertIn(SCRIPTS.signoff, replies[1]["text"])


class TestIdTiming(unittest.TestCase):
    def test_id_appended_on_first_transmission(self):
        records = run_session(one_utterance_events(), ["whiskey five tango sierra uniform hi"])
        reply = next(r for r in records if r["event"] == "reply")
        self.assertIn(SCRIPTS.id_text, reply["text"])

    def test_no_id_appended_on_second_transmission_within_window(self):
        clock = FakeClock()
        events = one_utterance_events() * 2
        records = run_session(
            events,
            ["whiskey five tango sierra uniform one", "whiskey five tango sierra uniform two"],
            clock=clock,
        )
        replies = [r for r in records if r["event"] == "reply"]
        self.assertEqual(len(replies), 2)
        self.assertIn(SCRIPTS.id_text, replies[0]["text"])
        self.assertNotIn(SCRIPTS.id_text, replies[1]["text"])


class TestSixtySecondCap(unittest.TestCase):
    def test_over_budget_reply_is_capped_and_synthesizer_reflects_it(self):
        budgets = BudgetsConfig(max_tx_seconds=3)
        # Four 5-word sentences (0.3s/word) plus the prepended ID: ~7.2s
        # total, over the 3s cap, but the ID plus the first sentence alone
        # (2.7s) fits once the trailing sentences are dropped.
        long_text = (
            "One two three four five. Six seven eight nine ten. "
            "Eleven twelve thirteen fourteen fifteen. Sixteen seventeen eighteen nineteen twenty."
        )
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(budgets),
                stream=ScriptedStream(one_utterance_events() + [RxEvent("dead")]),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=FakeBrain(long_text),
                synthesizer=FakeSynthesizer(seconds_per_word=0.3),
                player=FakePlayer(),
                clocks=Clocks(budgets, clock=FakeClock()),
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
        reply = next(r for r in records if r["event"] == "reply")
        self.assertLessEqual(reply["duration"], 3.0)
        self.assertIn(SCRIPTS.id_text, reply["text"])
        self.assertIn("One two three four five.", reply["text"])
        self.assertNotIn("twenty", reply["text"])


class TestDisarmOnFailedPreTxCheck(unittest.TestCase):
    def test_frequency_change_disarms_and_skips_transmission(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14236000, "USB", False),  # drifted
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        transmitter = FakeTransmitter()
        records = run_session(
            one_utterance_events(),
            ["whiskey five tango sierra uniform hi"],
            armed=True,
            pretx_check=check,
            transmitter=transmitter,
        )
        self.assertEqual(transmitter.sent, [])
        self.assertTrue(any(r["event"] == "disarm" for r in records))
        disarm = next(r for r in records if r["event"] == "disarm")
        self.assertEqual(disarm["reason"], "frequency-changed")

    def test_passing_check_transmits_when_armed(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14230000, "USB", False),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        transmitter = FakeTransmitter()
        records = run_session(
            one_utterance_events(),
            ["whiskey five tango sierra uniform hi"],
            armed=True,
            pretx_check=check,
            transmitter=transmitter,
        )
        self.assertEqual(len(transmitter.sent), 1)
        self.assertEqual(len([r for r in records if r["event"] == "disarm"]), 0)


class TestTxAnomalyDisarm(unittest.TestCase):
    def test_unconfirmed_unkey_disarms_after_sending(self):
        class FlakyTransmitter:
            def __init__(self):
                self.sent = []

            def send_async(self, wav_path):
                self.sent.append(str(wav_path))

            def is_done(self):
                return True

            def result(self):
                return TxResult(exit_code=None, saw_tx_off=False)  # never confirmed

        transmitter = FlakyTransmitter()
        records = run_session(
            one_utterance_events(),
            ["whiskey five tango sierra uniform hi"],
            armed=True,
            transmitter=transmitter,
        )
        self.assertEqual(len(transmitter.sent), 1)  # it did attempt to transmit
        disarm = next(r for r in records if r["event"] == "disarm")
        self.assertEqual(disarm["reason"], "tx-anomaly")


class TestSessionExpiry(unittest.TestCase):
    def test_expired_session_sends_one_grace_signoff_then_disarms(self):
        budgets = BudgetsConfig(armed_session_seconds=5)
        clock = FakeClock()
        clocks = Clocks(budgets, clock=clock)  # session_start captured at t=0
        clock.advance(6)  # now past armed_session_seconds
        baseline = RadioStatus(freq=1, mode="USB", tx=False)
        check = PreTxCheck(status_fn=lambda: baseline, rx_healthy_fn=lambda: True, baseline=baseline)
        transmitter = FakeTransmitter()

        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            session = Session(
                cfg(budgets),
                stream=ScriptedStream(one_utterance_events() + [RxEvent("dead")]),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=FakeBrain(),
                synthesizer=FakeSynthesizer(),
                player=FakePlayer(),
                transmitter=transmitter,
                armed=True,
                clocks=clocks,
                pretx_check=check,
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]

        self.assertEqual(len(transmitter.sent), 1)  # the grace transmission
        reply = next(r for r in records if r["event"] == "reply")
        self.assertEqual(reply["intent_source"], "session-expired")
        self.assertIn(SCRIPTS.signoff, reply["text"])
        self.assertTrue(any(r["event"] == "disarm" for r in records))


if __name__ == "__main__":
    unittest.main()
