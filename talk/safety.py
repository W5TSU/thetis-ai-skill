"""Session safety machinery: QSO/ID/session budget clocks, and the pre-
transmission radio-state guard. Pure logic driven by an injectable clock and
status function so it's deterministic to test without a real radio.
"""

from __future__ import annotations

import time
from dataclasses import dataclass

from talk.config import BudgetsConfig
from talk.constants import ID_INTERVAL_SECONDS


class Clocks:
    def __init__(self, budgets: BudgetsConfig, clock=time.monotonic):
        self._clock = clock
        self._budgets = budgets
        self.session_start = clock()
        self._qso_start: float | None = None
        self._last_activity: float | None = None
        self._last_id: float | None = None

    def note_triggered_turn(self) -> None:
        now = self._clock()
        if self._qso_start is None:
            self._qso_start = now
        self._last_activity = now

    def qso_elapsed(self) -> float | None:
        if self._qso_start is None:
            return None
        return self._clock() - self._qso_start

    def qso_over_budget(self) -> bool:
        elapsed = self.qso_elapsed()
        return elapsed is not None and elapsed >= self._budgets.max_qso_seconds

    def qso_idle_expired(self) -> bool:
        if self._last_activity is None:
            return False
        return self._clock() - self._last_activity >= self._budgets.qso_idle_close_seconds

    def close_qso(self) -> None:
        self._qso_start = None
        self._last_activity = None

    def needs_id(self) -> bool:
        if self._last_id is None:
            return True
        return self._clock() - self._last_id >= ID_INTERVAL_SECONDS

    def mark_id_sent(self) -> None:
        self._last_id = self._clock()

    def session_expired(self) -> bool:
        return self._clock() - self.session_start >= self._budgets.armed_session_seconds


@dataclass(frozen=True)
class RadioStatus:
    freq: int
    mode: str
    tx: bool


@dataclass(frozen=True)
class CheckResult:
    ok: bool
    reason: str | None = None


class PreTxCheck:
    """Guards every would-be transmission against radio state drifting out
    from under the armed session's original authorization.

    A CAT link that can't be reached fails closed when armed (disarm), but
    only warns in rehearsal — there's nothing to disarm and nothing gets
    transmitted either way.
    """

    def __init__(self, status_fn, rx_healthy_fn, baseline: RadioStatus | None):
        self._status_fn = status_fn
        self._rx_healthy_fn = rx_healthy_fn
        self.baseline = baseline

    def check(self, armed: bool) -> CheckResult:
        if not self._rx_healthy_fn():
            return CheckResult(False, "rx-unhealthy")

        status = self._status_fn()
        if status is None:
            return CheckResult(ok=not armed, reason="cat-unreachable")

        if self.baseline is not None:
            if status.freq != self.baseline.freq or status.mode != self.baseline.mode:
                return CheckResult(False, "frequency-changed")
            if status.tx:
                return CheckResult(False, "already-keyed")

        return CheckResult(True, None)
