"""Small signal-processing helpers shared across the audio pipeline."""

from __future__ import annotations

from array import array


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
