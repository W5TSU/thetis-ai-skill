"""dsp helpers: WAV sample-rate probing that doesn't assume integer PCM."""

import struct
import tempfile
import unittest
import wave
from pathlib import Path

from talk.dsp import wav_sample_rate

WAVE_FORMAT_IEEE_FLOAT = 3


def _float32_wav(path: Path, rate: int, *, extra_chunk: bytes = b"") -> None:
    """Write a minimal IEEE-float WAV, mirroring what `thetisctl tci
    rx-audio capture` emits (16-byte fmt chunk, format tag 3, no fact chunk).

    `extra_chunk` is spliced in before `data` to exercise chunk walking.
    """
    channels, bits = 2, 32
    body = struct.pack("<f", 0.0) * channels * 4  # 4 frames of silence
    block_align = channels * bits // 8
    fmt = struct.pack(
        "<HHIIHH",
        WAVE_FORMAT_IEEE_FLOAT,
        channels,
        rate,
        rate * block_align,
        block_align,
        bits,
    )
    chunks = b"fmt " + struct.pack("<I", len(fmt)) + fmt
    chunks += extra_chunk
    chunks += b"data" + struct.pack("<I", len(body)) + body
    riff = b"RIFF" + struct.pack("<I", 4 + len(chunks)) + b"WAVE" + chunks
    path.write_bytes(riff)


class TestWavSampleRate(unittest.TestCase):
    def test_reads_rate_from_float32_wav(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "float.wav"
            _float32_wav(p, 48000)
            self.assertEqual(wav_sample_rate(p), 48000)

    def test_reads_rate_from_integer_pcm_wav(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "pcm.wav"
            with wave.open(str(p), "wb") as w:
                w.setnchannels(1)
                w.setsampwidth(2)
                w.setframerate(22050)
                w.writeframes(b"\x00\x00" * 8)
            self.assertEqual(wav_sample_rate(p), 22050)

    def test_walks_past_chunks_before_fmt(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "junk-first.wav"
            # A JUNK chunk spliced ahead of `fmt ` -> a reader that assumes
            # fmt sits at a fixed offset would parse the wrong bytes.
            junk = b"JUNK" + struct.pack("<I", 4) + b"\x00\x00\x00\x00"
            _float32_wav(p, 24000)
            raw = p.read_bytes()
            p.write_bytes(raw[:12] + junk + raw[12:])
            self.assertEqual(wav_sample_rate(p), 24000)


if __name__ == "__main__":
    unittest.main()
