"""Session seam: RX events -> (VAD) -> transcribe -> wake match -> logged decision.

Uses a fake stream (pre-built events) and a fake transcriber — the Engines
seam — so this is fast, deterministic, and needs no models.
"""

import json
import pathlib
import tempfile
import unittest
from array import array
from dataclasses import dataclass

from talk.audio_rx import RxEvent
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


def tone(seconds, amplitude):
    n = int(seconds * RATE)
    return array("f", [amplitude if i % 2 else -amplitude for i in range(n)])


class FakeStream:
    """One burst of "speech" PCM, then a planned restart to end the run."""

    def __init__(self):
        self._events = [
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("audio", tone(0.6, 0.2)),
            RxEvent("audio", tone(1.0, 0.01)),
            RxEvent("dead"),
        ]

    def events(self):
        return iter(self._events)

    def stop(self):
        pass


@dataclass
class FakeTranscriber:
    text: str

    def transcribe(self, samples, sample_rate):
        return Transcript(text=self.text, no_speech_prob=0.0, avg_confidence=-0.1)


def make_cfg():
    return TalkConfig(
        radio=RadioConfig(host="x"),
        station=STATION,
        scripts=ScriptsConfig(id_text="id", signoff="clear"),
    )


class TestSessionRecognize(unittest.TestCase):
    def _run(self, transcriber):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            out = []
            session = Session(
                make_cfg(),
                stream=FakeStream(),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=out.append,
                transcriber=transcriber,
                station=STATION,
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
            return records, out

    def test_addressed_utterance_is_triggered_and_logged(self):
        records, out = self._run(FakeTranscriber("whiskey five tango sierra uniform go ahead"))
        u = next(r for r in records if r["event"] == "utterance")
        self.assertEqual(u["transcript"], "whiskey five tango sierra uniform go ahead")
        self.assertTrue(u["triggered"])
        self.assertEqual(u["trigger_kind"], "callsign")
        self.assertGreater(u["score"], 0)
        self.assertTrue(any("triggered" in line for line in out))

    def test_unrelated_speech_is_silent(self):
        records, out = self._run(FakeTranscriber("just some passing chatter on the band"))
        u = next(r for r in records if r["event"] == "utterance")
        self.assertFalse(u["triggered"])
        self.assertTrue(any("silent" in line for line in out))

    def test_no_transcriber_degrades_gracefully(self):
        records, out = self._run(transcriber=None)
        u = next(r for r in records if r["event"] == "utterance")
        self.assertNotIn("transcript", u)
        self.assertNotIn("triggered", u)


if __name__ == "__main__":
    unittest.main()
