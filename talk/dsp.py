"""Small signal-processing helpers shared across the audio pipeline."""

from __future__ import annotations

import struct
from array import array
from pathlib import Path


def wav_sample_rate(path: str | Path) -> int:
    """Sample rate (Hz) from a WAV file's ``fmt `` chunk, for any format tag.

    The stdlib ``wave`` module only reads integer PCM and raises on other
    format tags; ``thetisctl tci rx-audio capture`` writes IEEE-float
    (format 3) WAVs, so the sample-rate probe parses the RIFF header itself.
    Walks the chunk list rather than assuming ``fmt `` sits at a fixed
    offset.
    """
    with open(path, "rb") as f:
        header = f.read(12)
        if header[:4] != b"RIFF" or header[8:12] != b"WAVE":
            raise ValueError(f"{path}: not a RIFF/WAVE file")
        while True:
            chunk_header = f.read(8)
            if len(chunk_header) < 8:
                raise ValueError(f"{path}: no 'fmt ' chunk found")
            chunk_id, size = struct.unpack("<4sI", chunk_header)
            if chunk_id == b"fmt ":
                fmt = f.read(size)
                if len(fmt) < 8:
                    raise ValueError(f"{path}: truncated 'fmt ' chunk")
                # fmt body: wFormatTag(2) nChannels(2) nSamplesPerSec(4) ...
                return struct.unpack("<I", fmt[4:8])[0]
            f.seek(size + (size & 1), 1)  # RIFF chunks are word-aligned


def resample(samples: array, src_rate: int, dst_rate: int) -> array:
    """Linear-interpolation resample. Fine for speech; not for TX audio pacing."""
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
