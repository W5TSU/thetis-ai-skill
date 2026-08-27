"""Load and validate talk's station config from a TOML file.

The public seam is load(path) -> TalkConfig, raising ConfigError with a
message naming the offending key for any problem: missing file, bad TOML,
unknown keys, wrong types, or out-of-range values.
"""

from __future__ import annotations

import tomllib
from dataclasses import dataclass, field, fields
from pathlib import Path

VALID_SAMPLE_RATES = (8000, 12000, 24000, 48000)


class ConfigError(Exception):
    pass


@dataclass(frozen=True)
class RadioConfig:
    host: str
    cat_port: int = 13013
    tci_port: int = 50001
    rx: int = 0
    sample_rate: int = 24000


@dataclass(frozen=True)
class StationConfig:
    callsign: str
    phonetic_words: tuple[str, ...]
    wake_names: tuple[str, ...] = ()
    operator_name: str = ""
    qth: str = ""


@dataclass(frozen=True)
class ScriptsConfig:
    id_text: str
    signoff: str
    greeting: str = ""
    fallback_reply: str = "Please say again."
    signal_report_template: str = "Good copy on your signal."


@dataclass(frozen=True)
class BudgetsConfig:
    max_tx_seconds: int = 60
    max_qso_seconds: int = 600
    armed_session_seconds: int = 3600
    qso_idle_close_seconds: int = 120


@dataclass(frozen=True)
class VADConfig:
    threshold_ratio: float = 3.5
    hangover_ms: int = 800
    onset_ms: int = 100
    preroll_ms: int = 300
    min_utterance_ms: int = 400
    max_utterance_seconds: int = 30


@dataclass(frozen=True)
class ClaudeConfig:
    model: str = "claude-opus-5"
    max_context_turns: int = 12
    timeout_seconds: int = 20


@dataclass(frozen=True)
class LoggingConfig:
    dir: str = "talk/logs"
    enabled: bool = True


@dataclass(frozen=True)
class TalkConfig:
    radio: RadioConfig
    station: StationConfig
    scripts: ScriptsConfig
    budgets: BudgetsConfig = field(default_factory=BudgetsConfig)
    vad: VADConfig = field(default_factory=VADConfig)
    claude: ClaudeConfig = field(default_factory=ClaudeConfig)
    logging: LoggingConfig = field(default_factory=LoggingConfig)


_SECTIONS = {
    "radio": RadioConfig,
    "station": StationConfig,
    "scripts": ScriptsConfig,
    "budgets": BudgetsConfig,
    "vad": VADConfig,
    "claude": ClaudeConfig,
    "logging": LoggingConfig,
}
_REQUIRED_SECTIONS = ("radio", "station", "scripts")


def _build_section(name: str, cls, raw: dict):
    spec = {f.name: f for f in fields(cls)}
    for key in raw:
        if key not in spec:
            raise ConfigError(f"[{name}] has unknown key {key!r}")
    kwargs = {}
    for fname, f in spec.items():
        if fname in raw:
            value = raw[fname]
            if f.type in ("tuple[str, ...]",):
                if not isinstance(value, list) or not all(
                    isinstance(v, str) for v in value
                ):
                    raise ConfigError(f"[{name}] {fname} must be a list of strings")
                value = tuple(value)
            elif f.type == "str" and not isinstance(value, str):
                raise ConfigError(f"[{name}] {fname} must be a string")
            elif f.type == "int" and (isinstance(value, bool) or not isinstance(value, int)):
                raise ConfigError(f"[{name}] {fname} must be an integer")
            elif f.type == "float":
                if isinstance(value, bool) or not isinstance(value, (int, float)):
                    raise ConfigError(f"[{name}] {fname} must be a number")
                value = float(value)
            elif f.type == "bool" and not isinstance(value, bool):
                raise ConfigError(f"[{name}] {fname} must be true or false")
            kwargs[fname] = value
    try:
        return cls(**kwargs)
    except TypeError:
        missing = [n for n in spec if n not in kwargs]
        raise ConfigError(f"[{name}] missing required key(s): {', '.join(missing)}") from None


def _validate(cfg: TalkConfig) -> None:
    if cfg.radio.sample_rate not in VALID_SAMPLE_RATES:
        raise ConfigError(
            f"[radio] sample_rate must be one of {VALID_SAMPLE_RATES}, got {cfg.radio.sample_rate}"
        )
    if not cfg.radio.host:
        raise ConfigError("[radio] host must not be empty")
    if not cfg.station.callsign:
        raise ConfigError("[station] callsign must not be empty")
    if len(cfg.station.phonetic_words) < 2:
        raise ConfigError("[station] phonetic_words needs at least 2 words")
    for section, key in (("scripts", "id_text"), ("scripts", "signoff")):
        if not getattr(cfg.scripts, key).strip():
            raise ConfigError(f"[{section}] {key} must not be empty")
    for key in ("max_tx_seconds", "max_qso_seconds", "armed_session_seconds", "qso_idle_close_seconds"):
        if getattr(cfg.budgets, key) <= 0:
            raise ConfigError(f"[budgets] {key} must be positive")
    if cfg.budgets.max_tx_seconds > 60:
        raise ConfigError("[budgets] max_tx_seconds may not exceed 60")


def load(path: str | Path) -> TalkConfig:
    path = Path(path)
    try:
        raw = tomllib.loads(path.read_text())
    except OSError as e:
        raise ConfigError(f"cannot read config {path}: {e}") from e
    except tomllib.TOMLDecodeError as e:
        raise ConfigError(f"invalid TOML in {path}: {e}") from e

    for key in raw:
        if key not in _SECTIONS:
            raise ConfigError(f"unknown section or key {key!r} at top level")
    for name in _REQUIRED_SECTIONS:
        if name not in raw:
            raise ConfigError(f"missing required section [{name}]")

    sections = {}
    for name, cls in _SECTIONS.items():
        raw_section = raw.get(name, {})
        if not isinstance(raw_section, dict):
            raise ConfigError(f"[{name}] must be a table")
        if name in raw or name in _REQUIRED_SECTIONS:
            sections[name] = _build_section(name, cls, raw_section)
        else:
            sections[name] = cls()

    cfg = TalkConfig(**sections)
    _validate(cfg)
    return cfg
