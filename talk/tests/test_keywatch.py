"""KillSwitch seam: triggered() polls for a keypress or a delivered SIGINT.

Core polling logic is tested against a pipe standing in for stdin (select()
works on any readable fd) with tty_required=False, since this sandbox has no
real TTY. The TTY requirement itself is tested separately.
"""

import os
import signal
import unittest

from talk.keywatch import KillSwitch


class TestKillSwitchPolling(unittest.TestCase):
    def setUp(self):
        self.read_fd, self.write_fd = os.pipe()
        self.read_end = os.fdopen(self.read_fd, "rb", buffering=0)
        self.addCleanup(self.read_end.close)
        self.addCleanup(lambda: os.close(self.write_fd))

    def test_not_triggered_with_nothing_pending(self):
        with KillSwitch(stream=self.read_end, tty_required=False) as ks:
            self.assertFalse(ks.triggered())

    def test_triggered_once_a_byte_is_available(self):
        with KillSwitch(stream=self.read_end, tty_required=False) as ks:
            os.write(self.write_fd, b"x")
            self.assertTrue(ks.triggered())

    def test_triggered_latches_true(self):
        with KillSwitch(stream=self.read_end, tty_required=False) as ks:
            os.write(self.write_fd, b"x")
            ks.triggered()
            self.assertTrue(ks.triggered())  # stays triggered on later polls

    def test_sigint_triggers(self):
        with KillSwitch(stream=self.read_end, tty_required=False) as ks:
            self.assertFalse(ks.triggered())
            os.kill(os.getpid(), signal.SIGINT)
            self.assertTrue(ks.triggered())

    def test_exit_restores_previous_sigint_handler(self):
        original = signal.getsignal(signal.SIGINT)
        with KillSwitch(stream=self.read_end, tty_required=False):
            self.assertNotEqual(signal.getsignal(signal.SIGINT), original)
        self.assertEqual(signal.getsignal(signal.SIGINT), original)


class TestTtyRequirement(unittest.TestCase):
    def test_refuses_a_non_tty_stream_by_default(self):
        r, w = os.pipe()
        stream = os.fdopen(r, "rb", buffering=0)
        try:
            with self.assertRaises(RuntimeError):
                KillSwitch(stream=stream)  # tty_required defaults True
        finally:
            stream.close()
            os.close(w)


if __name__ == "__main__":
    unittest.main()
