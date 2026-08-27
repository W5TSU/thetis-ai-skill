"""RxStream seam: child lifecycle events — audio, planned restart, death, stall."""

import sys
import unittest

from talk.audio_rx import RxStream

# Fake stream children, in place of `thetisctl tci rx-audio stream`.
EMIT_THEN_EXIT0 = (
    "import struct, sys;"
    "sys.stdout.buffer.write(struct.pack('<4f', 0.1, -0.1, 0.2, -0.2));"
    "sys.stdout.buffer.flush()"
)
EXIT_1 = "import sys; sys.exit(1)"
HANG_SILENTLY = "import time; time.sleep(60)"


def cmd(snippet):
    return [sys.executable, "-c", snippet]


def collect(stream, stop_after, limit=50):
    """Iterate events until `stop_after` says stop or `limit` events pass."""
    events = []
    for ev in stream.events():
        events.append(ev)
        if stop_after(events) or len(events) >= limit:
            stream.stop()
            break
    return events


class TestRxStream(unittest.TestCase):
    def test_audio_decoded_then_planned_restart(self):
        stream = RxStream(cmd(EMIT_THEN_EXIT0), stall_timeout=5.0)
        events = collect(stream, lambda evs: sum(e.kind == "restart" for e in evs) >= 2)
        audio = [e for e in events if e.kind == "audio"]
        self.assertTrue(audio)
        self.assertAlmostEqual(audio[0].samples[0], 0.1, places=5)
        self.assertAlmostEqual(audio[0].samples[3], -0.2, places=5)
        # exit 0 = planned expiry: the stream restarts and keeps producing audio
        kinds = [e.kind for e in events]
        self.assertIn("restart", kinds)
        self.assertGreater(kinds.index("restart"), 0)
        self.assertEqual(sum(e.kind == "dead" for e in events), 0)

    def test_nonzero_exit_is_dead_and_ends(self):
        stream = RxStream(cmd(EXIT_1), stall_timeout=5.0)
        events = list(stream.events())
        self.assertEqual([e.kind for e in events], ["dead"])

    def test_silent_hang_is_stalled(self):
        stream = RxStream(cmd(HANG_SILENTLY), stall_timeout=0.3)
        events = collect(stream, lambda evs: any(e.kind == "stalled" for e in evs))
        self.assertIn("stalled", [e.kind for e in events])

    def test_stall_watchdog_can_be_suspended(self):
        stream = RxStream(cmd(HANG_SILENTLY), stall_timeout=0.3)
        stream.set_stall_suspended(True)
        events = []
        import threading, time

        def run():
            for ev in stream.events():
                events.append(ev)

        t = threading.Thread(target=run, daemon=True)
        t.start()
        time.sleep(1.0)  # several stall timeouts pass while suspended
        stream.stop()
        t.join(timeout=5)
        self.assertNotIn("stalled", [e.kind for e in events])


if __name__ == "__main__":
    unittest.main()
