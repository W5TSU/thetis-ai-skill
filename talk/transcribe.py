"""Local speech-to-text: resample the RX stream rate to Whisper's 16 kHz and
transcribe. The faster-whisper import is deferred into Transcriber.__init__
so the resampler and the rest of the test suite never need the model wheels.
"""

from __future__ import annotations

from array import array
from dataclasses import dataclass

WHISPER_RATE = 16000


@dataclass(frozen=True)
class Transcript:
    text: str
    no_speech_prob: float
    avg_confidence: float


def resample(samples: array, src_rate: int, dst_rate: int) -> array:
    """Linear-interpolation resample. Fine for speech STT input; not for TX audio."""
    if src_rate == dst_rate or len(samples) == 0:
        return array("f", samples)
    src_n = len(samples)
    dst_n = round(src_n * dst_rate / src_rate)
    if dst_n <= 0:
        return array("f")
    out = array("f", [0.0]) * dst_n
    scale = (src_n - 1) / (dst_n - 1) if dst_n > 1 else 0.0
    for i in range(dst_n):
        pos = i * scale
        lo = int(pos)
        hi = min(lo + 1, src_n - 1)
        frac = pos - lo
        out[i] = samples[lo] * (1 - frac) + samples[hi] * frac
    return out


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
