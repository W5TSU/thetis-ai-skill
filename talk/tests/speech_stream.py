"""Fake `thetisctl tci rx-audio stream` child that emits real synthesized
speech, for exercising talk's wake-match -> reply path end to end.

Unlike fake_stream.py (a bare tone that transcribes to nothing), this
renders a phrase with Piper so Whisper has something to transcribe and the
matcher something to trigger on.

    python -m talk.tests.speech_stream <rate_hz> <exit_code> "<phrase>"

Writes raw float32 LE mono PCM to stdout: 1.0 s noise, the phrase, 3.0 s
noise (enough trailing silence for the VAD hangover to close the
utterance), then exits <exit_code> (1 = "radio died", as RxStream expects
for a stream that ends).

Needs Piper + the lessac voice; only invoked by TALK_MODELS_DIR-gated
tests. Model dir: $TALK_MODELS_DIR, else talk/models/.
"""

import os
import sys
import wave
from pathlib import Path

import numpy as np

REPO_ROOT = Path(__file__).resolve().parents[2]


def _voice_path() -> str:
    base = os.environ.get("TALK_MODELS_DIR") or str(REPO_ROOT / "talk" / "models")
    return f"{base}/piper/en_US-lessac-medium.onnx"


def render(rate: int, phrase: str) -> np.ndarray:
    sys.path.insert(0, str(REPO_ROOT))
    from talk.tts import Synthesizer  # deferred: heavy, models-only

    tmp = REPO_ROOT / "talk" / "tests" / f".speech_stream_{os.getpid()}.wav"
    try:
        Synthesizer(_voice_path()).synthesize(phrase, tmp)
        with wave.open(str(tmp)) as w:
            assert w.getframerate() == 48000 and w.getsampwidth() == 2 and w.getnchannels() == 1
            pcm = np.frombuffer(w.readframes(w.getnframes()), dtype="<i2").astype(np.float32) / 32768.0
    finally:
        tmp.unlink(missing_ok=True)

    n_out = int(round(len(pcm) * rate / 48000))
    speech = np.interp(np.linspace(0, len(pcm) - 1, n_out), np.arange(len(pcm)), pcm).astype(np.float32)
    speech = (speech / (np.abs(speech).max() or 1.0) * 0.25).astype(np.float32)

    rng = np.random.default_rng(1)
    lead = (rng.standard_normal(int(1.0 * rate)) * 0.008).astype(np.float32)
    trail = (rng.standard_normal(int(3.0 * rate)) * 0.008).astype(np.float32)
    return np.concatenate([lead, speech, trail]).astype("<f4")


def main(argv: list[str]) -> int:
    rate, exit_code, phrase = int(argv[1]), int(argv[2]), argv[3]
    data = render(rate, phrase).tobytes()
    for i in range(0, len(data), 16384):
        sys.stdout.buffer.write(data[i : i + 16384])
        sys.stdout.buffer.flush()
    return exit_code


if __name__ == "__main__":
    sys.exit(main(sys.argv))
