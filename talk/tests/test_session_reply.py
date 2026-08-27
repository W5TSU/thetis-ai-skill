"""Session seam, rehearsal reply wiring: triggered utterance -> rule reply ->
synthesized WAV -> local playback. No TX-capable command is ever invoked.
"""

import json
import pathlib
import tempfile
import unittest
from array import array

from talk.audio_rx import RxEvent
from talk.config import RadioConfig, ScriptsConfig, StationConfig, TalkConfig, VADConfig
from talk.qsolog import SessionLog
from talk.rules import RuleEngine
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


class FakeStream:
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


class FakeTranscriber:
    def __init__(self, text):
        self.text = text

    def transcribe(self, samples, sample_rate):
        return Transcript(text=self.text, no_speech_prob=0.0, avg_confidence=-0.1)


class FakeSynthesizer:
    def __init__(self):
        self.calls = []

    def synthesize(self, text, out_path):
        self.calls.append(text)
        pathlib.Path(out_path).write_bytes(b"FAKEWAV")
        return 1.5


class FakePlayer:
    def __init__(self):
        self.played = []

    def play(self, wav_path):
        self.played.append(str(wav_path))


class FakeTransmitter:
    """Must never be called in rehearsal mode."""

    def send(self, wav_path):
        raise AssertionError("transmit invoked during rehearsal")


def make_cfg():
    return TalkConfig(radio=RadioConfig(host="x"), station=STATION, scripts=SCRIPTS)


class TestSessionReply(unittest.TestCase):
    def _run(self, text):
        with tempfile.TemporaryDirectory() as tmp:
            log = SessionLog(tmp, enabled=True)
            out = []
            synth = FakeSynthesizer()
            player = FakePlayer()
            session = Session(
                make_cfg(),
                stream=FakeStream(),
                vad=EnergyVAD(VADConfig(), RATE),
                log=log,
                sample_rate=RATE,
                out=out.append,
                transcriber=FakeTranscriber(text),
                station=STATION,
                rule_engine=RuleEngine(SCRIPTS, operator_name="Mark", qth="Oklahoma City"),
                synthesizer=synth,
                player=player,
                transmitter=FakeTransmitter(),
                armed=False,
            )
            session.run()
            records = []
            for p in pathlib.Path(tmp).iterdir():
                records += [json.loads(l) for l in (p / "session.jsonl").read_text().splitlines()]
            return records, out, synth, player

    def test_rule_match_is_synthesized_and_played_locally(self):
        records, out, synth, player = self._run(
            "whiskey five tango sierra uniform how do you copy"
        )
        reply = next(r for r in records if r["event"] == "reply")
        self.assertEqual(reply["intent"], "signal_report")
        self.assertEqual(reply["text"], "Good copy on your signal.")
        self.assertFalse(reply["armed"])
        self.assertEqual(synth.calls, ["Good copy on your signal."])
        self.assertEqual(len(player.played), 1)

    def test_unmatched_addressed_utterance_uses_fallback(self):
        records, out, synth, player = self._run(
            "whiskey five tango sierra uniform what's the weather like"
        )
        reply = next(r for r in records if r["event"] == "reply")
        self.assertEqual(reply["intent"], "fallback")
        self.assertEqual(reply["text"], SCRIPTS.fallback_reply)

    def test_silent_utterance_gets_no_reply(self):
        records, out, synth, player = self._run("just passing chatter on the band")
        self.assertFalse(any(r["event"] == "reply" for r in records))
        self.assertEqual(synth.calls, [])
        self.assertEqual(player.played, [])


if __name__ == "__main__":
    unittest.main()
