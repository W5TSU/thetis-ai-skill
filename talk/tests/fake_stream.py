"""Fake `thetisctl tci rx-audio stream` child for end-to-end tests.

Emits float32 LE PCM at 24 kHz sample-count (no pacing): 1 s noise,
0.8 s speech-loud tone, 1.5 s noise — then exits with the code given as
argv[1] (default 1, i.e. "radio died")."""

import struct
import sys

RATE = 24000


def emit(seconds, amplitude):
    n = int(seconds * RATE)
    frames = [amplitude if i % 2 else -amplitude for i in range(n)]
    sys.stdout.buffer.write(struct.pack(f"<{n}f", *frames))


emit(1.0, 0.01)
emit(0.8, 0.2)
emit(1.5, 0.01)
sys.stdout.buffer.flush()
sys.exit(int(sys.argv[1]) if len(sys.argv) > 1 else 1)
