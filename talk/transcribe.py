"""Local speech-to-text: resample the RX stream rate to Whisper's 16 kHz and
transcribe. The faster-whisper import is deferred into Transcriber.__init__
so the resampler and the rest of the test suite never need the model wheels.
"""

from __future__ import annotations

from array import array
from dataclasses import dataclass

from talk.dsp import resample

WHISPER_RATE = 16000


@dataclass(frozen=True)
class Transcript:
    text: str
    no_speech_prob: float
    avg_confidence: float


class Transcriber:
    def __init__(self, models_dir: str, model_size: str = "small"):
        from faster_whisper import WhisperModel  # deferred: heavy, optional at import time

        self._model = WhisperModel(model_size, download_root=models_dir, compute_type="int8")

    def transcribe(self, samples: array, sample_rate: int) -> Transcript:
        import numpy as np  # deferred alongside faster_whisper

        pcm = resample(samples, sample_rate, WHISPER_RATE) if sample_rate != WHISPER_RATE else samples
        segments, info = self._model.transcribe(np.frombuffer(pcm, dtype=np.float32), language="en")
        segments = list(segments)
        text = " ".join(s.text.strip() for s in segments).strip()
        avg_conf = (
            sum(getattr(s, "avg_logprob", 0.0) for s in segments) / len(segments)
            if segments
            else 0.0
        )
        return Transcript(
            text=text,
            no_speech_prob=getattr(info, "language_probability", 0.0),
            avg_confidence=avg_conf,
        )
