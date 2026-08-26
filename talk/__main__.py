"""Entry point: python -m talk --config <file> [--check] [--no-log]

Rehearsal mode is the default: the pipeline never keys the radio. Arming
(--armed plus the confirm phrase) is added by a later slice; until then this
entry validates config and prints the station banner.
"""

from __future__ import annotations

import argparse
import sys

from talk import config as config_mod
from talk.constants import ID_INTERVAL_SECONDS


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="talk", description="AI voice operator for a Thetis station")
    p.add_argument("--config", required=True, help="path to station config TOML")
    p.add_argument("--check", action="store_true", help="validate config, print banner, exit")
    p.add_argument("--no-log", action="store_true", help="disable session logging")
    return p


def banner(cfg: config_mod.TalkConfig) -> str:
    lines = [
        f"talk — AI voice operator  [REHEARSAL MODE — radio will not be keyed]",
        f"  station : {cfg.station.callsign} ({cfg.station.operator_name or 'operator unnamed'})",
        f"  radio   : {cfg.radio.host}  cat:{cfg.radio.cat_port} tci:{cfg.radio.tci_port}  rx{cfg.radio.rx} @ {cfg.radio.sample_rate} Hz",
        f"  wakes   : {' '.join(cfg.station.phonetic_words)} | {', '.join(cfg.station.wake_names) or '(none)'}",
        f"  budgets : tx<={cfg.budgets.max_tx_seconds}s  qso<={cfg.budgets.max_qso_seconds}s  session<={cfg.budgets.armed_session_seconds}s  id every {ID_INTERVAL_SECONDS}s",
        f"  logging : {'on -> ' + cfg.logging.dir if cfg.logging.enabled else 'off'}",
    ]
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        cfg = config_mod.load(args.config)
    except config_mod.ConfigError as e:
        print(f"talk: config error: {e}", file=sys.stderr)
        return 2
    print(banner(cfg))
    if args.check:
        return 0
    print("talk: the listening loop arrives in a later slice; use --check for now", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
