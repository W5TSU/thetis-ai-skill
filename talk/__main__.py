"""Entry point: python -m talk --config <file> [--check] [--no-log]

Rehearsal mode is the default: the pipeline never keys the radio. Arming
(--armed plus the confirm phrase) is added by a later slice.

TALK_STREAM_CMD (env) replaces the thetisctl RX stream command — the test
and debug hook at the Radio seam; when set, no thetisctl startup calls are
made and the configured sample rate is trusted as-is.
"""

from __future__ import annotations

import argparse
import os
import shlex
import shutil
import subprocess
import sys
import tempfile
import wave
from pathlib import Path

from talk import config as config_mod
from talk.audio_rx import RxStream
from talk.constants import ID_INTERVAL_SECONDS
from talk.qsolog import SessionLog
from talk.session import Session
from talk.vad import EnergyVAD

REPO_ROOT = Path(__file__).resolve().parents[1]


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
    return listen(cfg, no_log=args.no_log)


def thetisctl_path() -> str:
    found = shutil.which("thetisctl")
    if found:
        return found
    local = REPO_ROOT / "thetisctl"
    if local.exists():
        return str(local)
    print("talk: thetisctl not found on PATH or in the repo root", file=sys.stderr)
    raise SystemExit(2)


def probe_sample_rate(cfg: config_mod.TalkConfig, ctl: str) -> int:
    """Set the RX stream rate, then verify what Thetis actually delivers.

    The stream's stdout carries no rate metadata, but a capture WAV's header
    is written from the authoritative TCI frame headers — so a 1s probe
    capture tells us the real rate to run the VAD and resampler at.
    """
    tci = [ctl, "tci", "--host", cfg.radio.host, "--port", str(cfg.radio.tci_port)]
    subprocess.run(
        tci + ["audio-samplerate", str(cfg.radio.sample_rate)],
        check=True,
        capture_output=True,
        timeout=30,
    )
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
        probe = f.name
    subprocess.run(
        tci + ["rx-audio", "capture", str(cfg.radio.rx), "--duration", "1s", "--out", probe],
        check=True,
        capture_output=True,
        timeout=30,
    )
    with wave.open(probe) as w:
        actual = w.getframerate()
    os.unlink(probe)
    if actual != cfg.radio.sample_rate:
        print(
            f"talk: WARNING radio delivers {actual} Hz, not the requested "
            f"{cfg.radio.sample_rate} Hz; using {actual}",
            file=sys.stderr,
        )
    return actual


def listen(cfg: config_mod.TalkConfig, no_log: bool) -> int:
    override = os.environ.get("TALK_STREAM_CMD")
    if override:
        stream_cmd = shlex.split(override)
        rate = cfg.radio.sample_rate
    else:
        ctl = thetisctl_path()
        try:
            rate = probe_sample_rate(cfg, ctl)
        except subprocess.SubprocessError as e:
            print(f"talk: radio startup failed: {e}", file=sys.stderr)
            return 1
        stream_cmd = [
            ctl, "tci", "--host", cfg.radio.host, "--port", str(cfg.radio.tci_port),
            "rx-audio", "stream", str(cfg.radio.rx), "--duration", "4h",
        ]

    log = SessionLog(cfg.logging.dir, enabled=cfg.logging.enabled and not no_log)
    if log.enabled:
        print(f"session log: {log.dir}")
    session = Session(
        cfg,
        stream=RxStream(stream_cmd),
        vad=EnergyVAD(cfg.vad, rate),
        log=log,
        sample_rate=rate,
    )
    return session.run()


if __name__ == "__main__":
    sys.exit(main())
