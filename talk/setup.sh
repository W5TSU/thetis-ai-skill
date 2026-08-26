#!/usr/bin/env bash
# Bootstrap the talk voice-operator environment: venv, deps, speech models.
# Safe to re-run; everything lands under talk/.venv and talk/models (gitignored).
set -euo pipefail

cd "$(dirname "$0")"

# faster-whisper (CTranslate2) and piper-tts (onnxruntime) publish wheels for
# CPython up to 3.13 before they cover brand-new releases; prefer an older
# interpreter when several exist.
PY=""
for cand in python3.12 python3.13 python3.11 python3; do
    if command -v "$cand" >/dev/null 2>&1; then PY="$cand"; break; fi
done
[ -n "$PY" ] || { echo "setup: no python3 found" >&2; exit 1; }
echo "setup: using $($PY --version)"

if [ ! -d .venv ]; then
    "$PY" -m venv .venv
fi
# shellcheck disable=SC1091
source .venv/bin/activate

if ! pip install --quiet -r requirements.txt; then
    cat >&2 <<'EOF'
setup: pip install failed.
Most likely cause: no prebuilt wheels for this Python version (faster-whisper
needs CTranslate2, piper-tts needs onnxruntime). Install Python 3.12 and
re-run this script — it prefers the oldest supported interpreter it finds.
EOF
    exit 1
fi

echo "setup: pre-downloading Whisper small model (first run only)..."
python - <<'EOF'
from faster_whisper import WhisperModel
WhisperModel("small", download_root="models/whisper", compute_type="int8")
print("setup: whisper model ready")
EOF

VOICE="${TALK_PIPER_VOICE:-en_US-lessac-medium}"
echo "setup: downloading Piper voice $VOICE (first run only)..."
python -m piper.download_voices "$VOICE" --data-dir models/piper

if ! command -v aplay >/dev/null 2>&1; then
    echo "setup: WARNING — aplay not found; install alsa-utils for rehearsal playback" >&2
fi

echo "setup: done. Copy config.toml.example to config.toml, edit it, then:"
echo "  .venv/bin/python -m talk --config talk/config.toml --check   (from the repo root)"
