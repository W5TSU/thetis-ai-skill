#!/usr/bin/env bash
# Gate for the human-only armed live test — mirrors txlive_test.go's
# THETIS_LIVE_ALLOW_TX pattern. This script NEVER arms talk itself; it only
# prints the arming command for the human operator to type/run themselves,
# after checking the two independent opt-ins that make that deliberate.
#
# An agent must never run this script expecting it to transmit, and must
# never run the command it prints. See live_armed.md for the full
# procedure and .claude/skills/thetis-control/SKILL.md §6 for why this
# carve-out exists at all.
set -euo pipefail

CONFIRM_PHRASE="I-UNDERSTAND-THIS-KEYS-THE-RADIO"

if [ -z "${TALK_HOST:-}" ]; then
    echo "live_armed.sh: set TALK_HOST=<radio-ip> to identify the station under test." >&2
    exit 1
fi

if [ "${THETIS_LIVE_ALLOW_TX:-}" != "$CONFIRM_PHRASE" ]; then
    echo "live_armed.sh: requires THETIS_LIVE_ALLOW_TX=$CONFIRM_PHRASE" >&2
    echo "in addition to TALK_HOST. This is intentional — see live_armed.md." >&2
    echo "Do not set this automatically; only a human operator opts in." >&2
    exit 1
fi

if [ ! -t 0 ]; then
    echo "live_armed.sh: refusing — stdin is not a TTY. talk --armed needs a" >&2
    echo "real terminal for its kill switch; run this interactively." >&2
    exit 1
fi

cd "$(dirname "$0")/../.."
CONFIG="${TALK_CONFIG:-talk/config.toml}"

cat <<EOF

Both opt-ins are present and stdin is a terminal. Read live_armed.md before
going further, if you haven't already — a dummy load, a second receiver,
and the keypress-kill drill all come BEFORE this command.

The command below is for YOU to run, not this script:

  cd $(pwd)
  talk/.venv/bin/python -m talk --config "$CONFIG" \\
      --armed --confirm-tx $CONFIRM_PHRASE

This script will not run it for you.
EOF
