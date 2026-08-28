"""Entry point: python -m talk --config <file> [--check] [--no-log]
                              [--armed --confirm-tx <phrase>]

Rehearsal mode is the default: the pipeline never keys the radio. Arming
requires both --armed and the exact confirm phrase, and refuses without a
real terminal — the kill switch it provides is the only thing standing
between an armed session and a stuck transmitter, so it must be able to
read keypresses. This mirrors thetisctl's own --confirm-tx ritual; an AI
agent must never construct this command itself — only a human, at their
own terminal, arms this station. See SKILL.md's session-armed carve-out.

TALK_STREAM_CMD (env) replaces the thetisctl RX stream command — the test
and debug hook at the Radio seam; when set, no thetisctl startup calls are
made for RX and the configured sample rate is trusted as-is. It has no
effect on arming: transmission and radio-state polling always go through
the real thetisctl.
"""

from __future__ import annotations

import argparse
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import wave
from pathlib import Path

from talk import __version__
from talk import config as config_mod
from talk.audio_rx import RxStream
from talk.brain import Brain
from talk.constants import CONFIRM_PHRASE, ID_INTERVAL_SECONDS
from talk.keywatch import KillSwitch
from talk.qsolog import SessionLog
from talk.rules import RuleEngine
from talk.safety import Clocks, PreTxCheck, RadioStatus
from talk.session import Session
from talk.transmit import Player, Transmitter
from talk.vad import EnergyVAD

REPO_ROOT = Path(__file__).resolve().parents[1]


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="talk", description="AI voice operator for a Thetis station")
    p.add_argument("--version", action="version", version=f"talk {__version__}")
    p.add_argument("--config", required=True, help="path to station config TOML")
    p.add_argument("--check", action="store_true", help="validate config, print banner, exit")
    p.add_argument("--no-log", action="store_true", help="disable session logging")
    p.add_argument("--armed", action="store_true", help="transmit for real (default: rehearsal only)")
    p.add_argument("--confirm-tx", default="", help="required with --armed: the exact confirm phrase")
    return p


def banner(cfg: config_mod.TalkConfig, armed: bool) -> str:
    mode = "ARMED — WILL TRANSMIT" if armed else "REHEARSAL MODE — radio will not be keyed"
    lines = [
        f"talk {__version__} — AI voice operator  [{mode}]",
        f"  station : {cfg.station.callsign} ({cfg.station.operator_name or 'operator unnamed'})",
        f"  radio   : {cfg.radio.host}  cat:{cfg.radio.cat_port} tci:{cfg.radio.tci_port}  rx{cfg.radio.rx} @ {cfg.radio.sample_rate} Hz",
        f"  wakes   : {' '.join(cfg.station.phonetic_words)} | {', '.join(cfg.station.wake_names) or '(none)'}",
        f"  budgets : tx<={cfg.budgets.max_tx_seconds}s  qso<={cfg.budgets.max_qso_seconds}s  session<={cfg.budgets.armed_session_seconds}s  id every {ID_INTERVAL_SECONDS}s",
        f"  logging : {'on -> ' + cfg.logging.dir if cfg.logging.enabled else 'off'}",
    ]
    if armed:
        lines.append("  kill    : press any key, or Ctrl-C, to unkey and disarm instantly")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        cfg = config_mod.load(args.config)
    except config_mod.ConfigError as e:
        print(f"talk: config error: {e}", file=sys.stderr)
        return 2

    if args.armed and args.confirm_tx != CONFIRM_PHRASE:
        print(
            "talk: --armed requires --confirm-tx with the exact confirm phrase "
            f"(see SKILL.md); refusing to guess or proceed without it.",
            file=sys.stderr,
        )
        return 2
    if args.armed and not sys.stdin.isatty():
        # Checked before any radio I/O: the kill switch that makes arming
        # survivable needs a real terminal, so there's no point going further.
        print("talk: --armed requires a real terminal (stdin is not a TTY)", file=sys.stderr)
        return 2

    print(banner(cfg, armed=args.armed))
    if args.check:
        return 0
    return listen(cfg, no_log=args.no_log, armed=args.armed)


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


_CAT_STATUS_RE = {
    "freq": re.compile(r"Freq:\s*(\d+)\s*Hz"),
    "mode": re.compile(r"Mode:\s*(\S+)"),
    "tx": re.compile(r"TX:\s*(true|false)", re.IGNORECASE),
}


def get_cat_status(ctl: str, cfg: config_mod.TalkConfig) -> RadioStatus | None:
    """Poll `thetisctl cat status`; None on any failure (fails closed by the
    caller — PreTxCheck treats a None status as cat-unreachable)."""
    try:
        result = subprocess.run(
            [ctl, "cat", "--host", cfg.radio.host, "--port", str(cfg.radio.cat_port), "status"],
            capture_output=True, text=True, timeout=10,
        )
    except subprocess.SubprocessError:
        return None
    if result.returncode != 0:
        return None
    values = {}
    for key, pattern in _CAT_STATUS_RE.items():
        m = pattern.search(result.stdout)
        if not m:
            return None
        values[key] = m.group(1)
    return RadioStatus(freq=int(values["freq"]), mode=values["mode"], tx=values["tx"].lower() == "true")


def tx_cmd_builder(ctl: str, cfg: config_mod.TalkConfig):
    def build(wav_path) -> list[str]:
        return [
            ctl, "tci", "--host", cfg.radio.host, "--port", str(cfg.radio.tci_port),
            "tx-audio", "send", str(cfg.radio.rx),
            "--file", str(wav_path),
            "--max-duration", f"{cfg.budgets.max_tx_seconds}s",
            "--confirm-tx", CONFIRM_PHRASE,
        ]

    return build


def listen(cfg: config_mod.TalkConfig, no_log: bool, armed: bool = False) -> int:
    override = os.environ.get("TALK_STREAM_CMD")
    ctl = None if override and not armed else thetisctl_path()
    if override:
        stream_cmd = shlex.split(override)
        rate = cfg.radio.sample_rate
    else:
        try:
            rate = probe_sample_rate(cfg, ctl)
        except subprocess.SubprocessError as e:
            print(f"talk: radio startup failed: {e}", file=sys.stderr)
            return 1
        stream_cmd = [
            ctl, "tci", "--host", cfg.radio.host, "--port", str(cfg.radio.tci_port),
            "rx-audio", "stream", str(cfg.radio.rx), "--duration", "4h",
        ]

    pretx_check = None
    transmitter = None
    kill_switch = None
    if armed:
        baseline = get_cat_status(ctl, cfg)
        if baseline is None:
            print(
                "talk: could not read radio status via CAT; refusing to arm "
                "(check the CAT server is enabled and reachable)",
                file=sys.stderr,
            )
            return 1
        print(f"arm baseline: {baseline.freq} Hz, {baseline.mode}, tx={baseline.tx}")
        pretx_check = PreTxCheck(
            status_fn=lambda: get_cat_status(ctl, cfg),
            rx_healthy_fn=lambda: True,
            baseline=baseline,
        )
        transmitter = Transmitter(cmd_builder=tx_cmd_builder(ctl, cfg))

    log = SessionLog(cfg.logging.dir, enabled=cfg.logging.enabled and not no_log)
    if log.enabled:
        print(f"session log: {log.dir}")

    transcriber = None
    try:
        from talk.transcribe import Transcriber

        transcriber = Transcriber(models_dir=str(REPO_ROOT / "talk" / "models" / "whisper"))
    except ImportError:
        print(
            "talk: WARNING speech-to-text unavailable (faster-whisper not installed); "
            "listening only, no wake recognition — run talk/setup.sh",
            file=sys.stderr,
        )

    synthesizer = None
    try:
        from talk.tts import Synthesizer

        voice = REPO_ROOT / "talk" / "models" / "piper" / "en_US-lessac-medium.onnx"
        synthesizer = Synthesizer(model_path=str(voice))
    except ImportError:
        print(
            "talk: WARNING text-to-speech unavailable (piper-tts not installed); "
            "will not synthesize replies — run talk/setup.sh",
            file=sys.stderr,
        )

    claude_client = None
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("talk: no ANTHROPIC_API_KEY set; running rules-only", file=sys.stderr)
    else:
        try:
            import anthropic

            claude_client = anthropic.Anthropic()
        except ImportError:
            print(
                "talk: WARNING anthropic not installed; running rules-only — run talk/setup.sh",
                file=sys.stderr,
            )

    brain = Brain(
        rule_engine=RuleEngine(cfg.scripts, cfg.station.operator_name, cfg.station.qth),
        scripts=cfg.scripts,
        station=cfg.station,
        claude_config=cfg.claude,
        claude_client=claude_client,
    )

    def build_session(kill_switch) -> Session:
        return Session(
            cfg,
            stream=RxStream(stream_cmd),
            vad=EnergyVAD(cfg.vad, rate),
            log=log,
            sample_rate=rate,
            transcriber=transcriber,
            brain=brain,
            synthesizer=synthesizer,
            player=Player(),
            transmitter=transmitter,
            armed=armed,
            clocks=Clocks(cfg.budgets),
            pretx_check=pretx_check,
            kill_switch=kill_switch,
        )

    if armed:
        # The KillSwitch is what makes an armed session survivable — it
        # requires a real TTY, matching the isatty check above.
        with KillSwitch() as ks:
            return build_session(ks).run()
    return build_session(kill_switch=None).run()


if __name__ == "__main__":
    sys.exit(main())
