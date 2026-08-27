"""Local text-to-speech, always rendered to 48000 Hz / mono / 16-bit — the
only WAV format proven against real TX hardware (see internal/tci/wav.go
and cmd/thetisctl/txlive_test.go). thetisctl does no resampling itself.

The piper import is deferred so the offline test suite needs no model
wheels.
"""

from __future__ import annotations

import re
import wave
from array import array
from pathlib import Path
from typing import Callable

from talk.dsp import resample

TX_RATE = 48000


def drop_trailing_sentences(
    text: str,
    seconds_per_sentence_estimate: Callable[[str], float],
    budget_seconds: float,
) -> str:
    """Drop trailing sentences until the estimate fits the budget.

    Used as a pre-synthesis shortening pass, and again after a real
    over-length synthesis as a re-synthesis fallback. A lone sentence that
    alone exceeds the budget is left as-is — refusal is decided by the
    caller once the actual WAV duration is known.
    """
    sentences = [s.strip() for s in re.split(r"(?<=[.!?])\s+", text.strip()) if s.strip()]
    if not sentences:
        return text
    while len(sentences) > 1:
        total = sum(seconds_per_sentence_estimate(s) for s in sentences)
        if total <= budget_seconds:
            break
        sentences.pop()
    return " ".join(sentences)


def synthesize_capped(
    synthesizer, text: str, out_path: str | Path, max_seconds: float
) -> tuple[float, bool, str]:
    """Synthesize, then shorten and re-synthesize if the result runs long.

    The common case (composed reply already under budget) costs one
    synthesis call. Over budget, sentences are dropped from the end and the
    remainder re-rendered; a single sentence that alone exceeds the budget
    is left as-is and reported as refused rather than silently truncated
    mid-word.
    """
    duration = synthesizer.synthesize(text, out_path)
    if duration <= max_seconds:
        return duration, False, text

    def estimate(sentence: str) -> float:
        probe = str(out_path) + ".probe.wav"
        return synthesizer.synthesize(sentence, probe)

    shortened = drop_trailing_sentences(text, estimate, max_seconds)
    duration = synthesizer.synthesize(shortened, out_path)
    return duration, duration > max_seconds, shortened


class Synthesizer:
    def __init__(self, model_path: str):
        from piper import PiperVoice  # deferred: heavy, optional at import time

        self._voice = PiperVoice.load(model_path)

    def synthesize(self, text: str, out_path: str | Path) -> float:
        """Render text to a TX-ready WAV at out_path; returns duration seconds."""
        pcm = array("f")
        native_rate = None
        for chunk in self._voice.synthesize(text):
            native_rate = chunk.sample_rate
            pcm.extend(chunk.audio_float_array.astype("float32").tolist())

        if native_rate is None:
            native_rate = TX_RATE  # nothing synthesized; write a silent stub
        resampled = resample(pcm, native_rate, TX_RATE)
        pcm16 = array(
            "h", (max(-32768, min(32767, int(s * 32767))) for s in resampled)
        )
        with wave.open(str(out_path), "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(2)
            w.setframerate(TX_RATE)
            w.writeframes(pcm16.tobytes())
        return len(resampled) / TX_RATE
