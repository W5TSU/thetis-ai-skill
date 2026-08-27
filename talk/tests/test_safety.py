"""Safety seam: Clocks (QSO/ID/session budgets) and PreTxCheck (radio-state
guard), both driven by an injectable clock/status so they're deterministic.
"""

import unittest

from talk.config import BudgetsConfig
from talk.safety import Clocks, PreTxCheck, RadioStatus


class FakeClock:
    def __init__(self, start=0.0):
        self.t = start

    def __call__(self):
        return self.t

    def advance(self, seconds):
        self.t += seconds


BUDGETS = BudgetsConfig(
    max_tx_seconds=60,
    max_qso_seconds=600,
    armed_session_seconds=3600,
    qso_idle_close_seconds=120,
)


class TestClocksQso(unittest.TestCase):
    def test_qso_not_open_until_first_triggered_turn(self):
        clocks = Clocks(BUDGETS, clock=FakeClock())
        self.assertIsNone(clocks.qso_elapsed())
        self.assertFalse(clocks.qso_over_budget())

    def test_qso_elapsed_tracks_from_first_turn(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.note_triggered_turn()
        clock.advance(30)
        self.assertAlmostEqual(clocks.qso_elapsed(), 30.0)

    def test_qso_over_budget_at_cap(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.note_triggered_turn()
        clock.advance(599)
        self.assertFalse(clocks.qso_over_budget())
        clock.advance(1)
        self.assertTrue(clocks.qso_over_budget())

    def test_close_qso_resets_elapsed(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.note_triggered_turn()
        clock.advance(100)
        clocks.close_qso()
        self.assertIsNone(clocks.qso_elapsed())

    def test_idle_expiry_after_no_activity(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.note_triggered_turn()
        clock.advance(119)
        self.assertFalse(clocks.qso_idle_expired())
        clock.advance(2)
        self.assertTrue(clocks.qso_idle_expired())

    def test_idle_expiry_false_when_no_qso_open(self):
        clocks = Clocks(BUDGETS, clock=FakeClock())
        self.assertFalse(clocks.qso_idle_expired())


class TestClocksId(unittest.TestCase):
    def test_needs_id_before_first_send(self):
        clocks = Clocks(BUDGETS, clock=FakeClock())
        self.assertTrue(clocks.needs_id())

    def test_no_id_needed_right_after_sending(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.mark_id_sent()
        self.assertFalse(clocks.needs_id())

    def test_needs_id_again_after_600_seconds(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clocks.mark_id_sent()
        clock.advance(599)
        self.assertFalse(clocks.needs_id())
        clock.advance(1)
        self.assertTrue(clocks.needs_id())


class TestClocksSession(unittest.TestCase):
    def test_session_expires_at_armed_session_seconds(self):
        clock = FakeClock()
        clocks = Clocks(BUDGETS, clock=clock)
        clock.advance(3599)
        self.assertFalse(clocks.session_expired())
        clock.advance(1)
        self.assertTrue(clocks.session_expired())


class TestPreTxCheck(unittest.TestCase):
    def test_ok_with_matching_baseline_and_healthy_rx(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14230000, "USB", False),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        result = check.check(armed=True)
        self.assertTrue(result.ok)
        self.assertIsNone(result.reason)

    def test_frequency_changed_fails_closed(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14236000, "USB", False),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        result = check.check(armed=True)
        self.assertFalse(result.ok)
        self.assertEqual(result.reason, "frequency-changed")

    def test_mode_changed_fails_closed(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14230000, "LSB", False),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        self.assertFalse(check.check(armed=True).ok)

    def test_already_keyed_fails_closed(self):
        baseline = RadioStatus(freq=14230000, mode="USB", tx=False)
        check = PreTxCheck(
            status_fn=lambda: RadioStatus(14230000, "USB", True),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        result = check.check(armed=True)
        self.assertFalse(result.ok)
        self.assertEqual(result.reason, "already-keyed")

    def test_unhealthy_rx_fails_closed(self):
        check = PreTxCheck(status_fn=lambda: None, rx_healthy_fn=lambda: False, baseline=None)
        result = check.check(armed=True)
        self.assertFalse(result.ok)
        self.assertEqual(result.reason, "rx-unhealthy")

    def test_cat_unreachable_fails_closed_when_armed(self):
        check = PreTxCheck(status_fn=lambda: None, rx_healthy_fn=lambda: True, baseline=None)
        result = check.check(armed=True)
        self.assertFalse(result.ok)
        self.assertEqual(result.reason, "cat-unreachable")

    def test_cat_unreachable_only_warns_in_rehearsal(self):
        check = PreTxCheck(status_fn=lambda: None, rx_healthy_fn=lambda: True, baseline=None)
        result = check.check(armed=False)
        self.assertTrue(result.ok)
        self.assertEqual(result.reason, "cat-unreachable")


if __name__ == "__main__":
    unittest.main()
