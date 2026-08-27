# Talk — an AI voice operator for the station

An AI that can *hear voice over the radio and respond by voice over the
radio* — an AI radio operator, supervised by a human control operator. All
eight implementation slices below have landed and are covered by the
offline test suite; **the one thing that has never happened is a real armed
transmission** — no radio was reachable during development, and arming is,
by design, a human-only act. See
[`tests/live_armed.md`](tests/live_armed.md) before ever running this
armed for the first time. Quick-start setup/config lives in the "talk —
AI voice operator" section of [the root README](../README.md); this file
is the design record and the fuller rationale behind it.

> **⚠️ Everything in the root README's transmit warning applies double here.**
> This feature keys a real transmitter repeatedly under a single human
> authorization. It is designed around a human control operator being
> physically present, with rehearsal (no-TX) mode as the default and hard
> airtime budgets — but it has not yet been proven against a real radio.
> Read `tests/live_armed.md` in full before ever passing `--armed`.

## What it does

While the operator sits at the control point, they tune the radio, then arm
the loop. It listens continuously to RX audio, detects utterances, and
transcribes them locally. When — and only when — an utterance contains a
*wake* (the station callsign in spoken phonetics, fuzzy-matched, or a
configured wake name), it composes a reply, synthesizes speech, and transmits
it. Everything else it hears is logged silently. Standard exchanges (signal
report, "are you there", name/QTH, repeat requests) are answered instantly by
canned rules; everything else goes to Claude under a strict persona prompt,
degrading gracefully to rules-only if the API is unreachable.

By default the whole pipeline runs in **rehearsal mode**: it listens,
transcribes, decides, and plays its would-be reply on the local speakers —
the radio is never keyed. Transmitting requires an explicit
`--armed --confirm-tx` arming ritual at a real terminal, and **any keypress
instantly unkeys and disarms**.

## The decisions already made

- **Role**: speaks only when addressed. It never answers CQs or joins other
  people's QSOs (that could grow later; not in v1).
- **Mode**: analog SSB/FM voice only. No FreeDV/RADE in this feature.
- **Supervision**: session-armed. The human at the terminal is the control
  operator; arming is a deliberate flagged act; rehearsal is the default
  posture; the terminal being open *is* the control point (no daemon mode,
  no remote kill — deliberately).
- **Budgets, enforced in code**: max 60 s per transmission, max 10 min per
  QSO (then a scripted sign-off), armed-session expiry, and instant disarm
  on: keypress, frequency/mode changed under the loop, RX stream death, or
  any transmit anomaly. Station ID timing (first TX, every 10 minutes, at
  sign-off) is hard-coded because it's regulatory, not stylistic; the ID
  *text* is operator-scripted config.
- **Frequency**: the operator owns the dial. The loop never retunes, and
  disarms if the frequency changes underneath it.
- **Stack**: Python orchestrator in this directory driving `thetisctl` child
  processes — the Go tool is unchanged. Local faster-whisper (CPU) for
  speech-to-text, local Piper for text-to-speech, Claude API for the
  conversational brain with a canned-rules fallback. Plain venv +
  `requirements.txt`; config in one TOML file.
- **Radio plumbing** (already exists in thetisctl): one long-running
  `tci rx-audio stream` child supplies continuous PCM; each reply is one
  `tci tx-audio send` invocation, which keys PTT itself and
  confirm-unkeys on completion, error, or interrupt.
- **Logging**: on by default — per-session directory with RX utterance WAVs,
  TX reply WAVs, and a JSONL transcript of every turn. These double as
  station records.
- **Testing seams**: two. A *Radio* boundary (PCM in, thetisctl commands
  out) and an *Engines* boundary (STT/TTS/Claude), so the entire turn
  pipeline is testable offline by injecting recorded audio and faking the
  engines. Live armed testing follows the repo's existing environment-gated,
  human-only procedure.

## Implementation, as landed

Built as tracer-bullet slices, each demoable on its own — all eight are
complete:

1. ✅ **Bootstrap & config** — venv/model setup, TOML config, station banner.
2. ✅ **Hear the band** — continuous RX streaming + voice-activity detection;
   utterance WAVs and turn records appear in the session log.
3. ✅ **Recognize being called** — local transcription + fuzzy wake matching.
4. ✅ **Reply in rehearsal** — rule engine + TTS; replies audible on local
   speakers, radio never keyed.
5. ✅ **Claude brain** — conversational replies with per-QSO context and live
   degradation to rules.
6. ✅ **Safety core** — the clocks, budgets, and disarm conditions, under a
   fake-clock test suite.
7. ✅ **Armed transmit** — the arming ritual, the keypress kill switch with
   confirmed unkey, real transmission, and the control skill's explicit
   session-armed carve-out (`SKILL.md` §6) — the first feature in this repo
   that keys repeatedly on one human authorization.
8. ✅ **Docs & glossary** — this file, the root README section, `AGENTS.md`'s
   Python-subtree contract, and the repo glossary.

Vocabulary (Utterance, Turn, Wake, QSO, Rehearsal mode, Armed session,
Disarm, ...) is defined once, canonically, in the
[repo's `CONTEXT.md`](../CONTEXT.md) — not duplicated here.
