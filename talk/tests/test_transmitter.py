"""Transmitter seam: spawn a tx-audio-send-shaped child, abort() via SIGINT
only (the child's own handler unkeys), never SIGKILL.
"""

import sys
import tempfile
import unittest
from pathlib import Path

from talk.transmit import Transmitter

# Fake tx-audio-send children, standing in for
# `thetisctl tci tx-audio send <rx> --file <wav> --max-duration ...`.
QUICK_SUCCESS = "import sys; print('TX ON'); print('TX OFF'); sys.exit(0)"
CONFIRMED_UNKEY_ON_SIGINT = (
    "import signal, sys, time\n"
    "def handler(sig, frame):\n"
    "    print('TX OFF (confirmed)')\n"
    "    sys.exit(130)\n"
    "signal.signal(signal.SIGINT, handler)\n"
    "print('TX ON')\n"
    "time.sleep(30)\n"
)
IGNORES_SIGINT_BRIEFLY = (
    "import signal, sys, time\n"
    "signal.signal(signal.SIGINT, signal.SIG_IGN)\n"
    "print('TX ON')\n"
    "time.sleep(0.3)\n"
    "print('TX OFF (late)')\n"
    "sys.exit(0)\n"
)


def cmd(snippet):
    return [sys.executable, "-c", snippet]


class TestTransmitter(unittest.TestCase):
    def test_send_runs_to_completion_and_reports_success(self):
        tx = Transmitter(cmd_builder=lambda wav: cmd(QUICK_SUCCESS))
        with tempfile.TemporaryDirectory() as tmp:
            wav = Path(tmp) / "reply.wav"
            wav.write_bytes(b"RIFF")
            result = tx.send(wav)
        self.assertEqual(result.exit_code, 0)
        self.assertTrue(result.saw_tx_off)

    def test_abort_sends_sigint_and_waits_for_confirmed_unkey(self):
        tx = Transmitter(cmd_builder=lambda wav: cmd(CONFIRMED_UNKEY_ON_SIGINT))
        with tempfile.TemporaryDirectory() as tmp:
            wav = Path(tmp) / "reply.wav"
            wav.write_bytes(b"RIFF")
            tx.send_async(wav)
            import time

            time.sleep(0.1)  # let it print "TX ON" and start sleeping
            result = tx.abort(grace_seconds=5)
        self.assertEqual(result.exit_code, 130)
        self.assertTrue(result.saw_tx_off)

    def test_abort_never_sends_sigkill_even_if_slow_to_unkey(self):
        tx = Transmitter(cmd_builder=lambda wav: cmd(IGNORES_SIGINT_BRIEFLY))
        with tempfile.TemporaryDirectory() as tmp:
            wav = Path(tmp) / "reply.wav"
            wav.write_bytes(b"RIFF")
            tx.send_async(wav)
            import time

            time.sleep(0.05)
            result = tx.abort(grace_seconds=5)
        # The child ignored SIGINT briefly but still exited on its own terms
        # (0, not killed) — abort() must have waited rather than escalating.
        self.assertEqual(result.exit_code, 0)
        self.assertTrue(result.saw_tx_off)

    def test_abort_with_nothing_in_flight_is_a_noop(self):
        tx = Transmitter(cmd_builder=lambda wav: cmd(QUICK_SUCCESS))
        self.assertIsNone(tx.abort(grace_seconds=1))


if __name__ == "__main__":
    unittest.main()
