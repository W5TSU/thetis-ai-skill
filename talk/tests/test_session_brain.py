"""Session seam: triggered utterances route through Brain (rules -> Claude ->
fallback), with per-QSO context carried across turns and logged sources.
"""

import json
import pathlib
import tempfile
import unittest
from array import array
from dataclasses import dataclass, field

from talk.audio_rx import RxEvent
from talk.brain import Brain, Decision
from talk.config import RadioConfig, ScriptsConfig, StationConfig, TalkConfig, VADConfig
from talk.qsolog import SessionLog
from talk.session import Session
from talk.transcribe import Transcript
from talk.vad import EnergyVAD

RATE = 24000
STATION = StationConfig(
    callsign="W5TSU",
    phonetic_words=("whiskey", "five", "tango", "sierra", "uniform"),
    wake_names=("thetis",),
)
SCRIPTS = ScriptsConfig(id_text="id", signoff="clear", fallback_reply="Please say again.")


def tone(seconds, amplitude):
    n = int(seconds * RATE)
    return array("f", [amplitude if i % 2 else -amplitude for i in range(n)])


def utterance_events(*texts):
    events = []
    for _ in texts:
        events += [
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("audio", tone(0.6, 0.2)),
            RxEvent("audio", tone(1.0, 0.01)),
        ]
    events.append(RxEvent("dead"))
    return events


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


@dataclass
class FakeBrain:
    """Records the qso object it was given each call, and its turn count."""

    decisions: list
    seen_qso_turns: list = field(default_factory=list)

    def compose(self, heard, qso):
        self.seen_qso_turns.append(len(qso.turns))
        d = self.decisions.pop(0)
        qso.add_turn(heard, d.reply_text)
        return d


class FakeSynthesizer:
    def synthesize(self, text, out_path):
        pathlib.Path(out_path).write_bytes(b"FAKE")
        return 1.0


class FakePlayer:
    def __init__(self):
        self.played = []

    def play(self, wav_path):
        self.played.append(str(wav_path))


def make_cfg():
    return TalkConfig(radio=RadioConfig(host="x"), station=STATION, scripts=SCRIPTS)


class TestSessionBrainWiring(unittest.TestCase):
    def test_brain_decision_source_is_logged(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            brain = FakeBrain(decisions=[Decision("Sure thing.", "claude", None)])
            session = Session(
                make_cfg(),
                stream=ScriptedStream(utterance_events("x")),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(["whiskey five tango sierra uniform hi"]),
                station=STATION,
                brain=brain,
                synthesizer=FakeSynthesizer(),
                player=FakePlayer(),
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
            reply = next(r for r in records if r["event"] == "reply")
            self.assertEqual(reply["intent_source"], "claude")
            self.assertEqual(reply["text"], "Sure thing.")

    def test_qso_context_accumulates_across_turns_in_one_run(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            brain = FakeBrain(
                decisions=[
                    Decision("first reply", "claude", None),
                    Decision("second reply", "claude", None),
                ]
            )
            session = Session(
                make_cfg(),
                stream=ScriptedStream(utterance_events("a", "b")),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=lambda *_: None,
                transcriber=ScriptedTranscriber(
                    [
                        "whiskey five tango sierra uniform first",
                        "whiskey five tango sierra uniform second",
                    ]
                ),
                station=STATION,
                brain=brain,
                synthesizer=FakeSynthesizer(),
                player=FakePlayer(),
            )
            session.run()
            # Second call sees the QSO context grown by the first turn.
            self.assertEqual(brain.seen_qso_turns, [0, 1])


if __name__ == "__main__":
    unittest.main()
