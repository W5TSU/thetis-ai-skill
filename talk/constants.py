"""Constants shared across the talk orchestrator."""

# Must match safety.ConfirmPhrase in internal/safety/txgate.go — the Go side is
# the source of truth; a unit test greps that file to catch drift.
CONFIRM_PHRASE = "I-UNDERSTAND-THIS-KEYS-THE-RADIO"

# Regulatory station-ID interval (47 CFR §97.119: every 10 minutes during a
# communication and at its end). Deliberately not configurable.
ID_INTERVAL_SECONDS = 600
