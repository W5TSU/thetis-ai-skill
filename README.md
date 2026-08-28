# thetisctl

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**thetisctl** provides remote control of a running Thetis SDR instance over its
existing network using CAT-over-TCP (frequency, mode, filters, VFO,
RIT/XIT, split, AGC, attenuator/preamp, band) and TCI-over-WebSocket (the
same controls plus RX/TX audio streaming, CW keying, and transmit). 

No Thetis-side code changes are required — both servers already exist in
Thetis, they're just usually off by default. ( > Version 1.3.17 )

> **⚠️ Only point this at a radio you directly control.** `thetisctl` can key
> a real transmitter into a real antenna — its TX-capable commands cause
> actual on-air RF, not a simulation. Only use it against a Thetis instance
> you personally own or operate, are licensed to transmit from, and can
> physically reach or shut down if something goes wrong. Neither of Thetis's
> control protocols has authentication, so anything reachable on the network
> is controllable — never run this against someone else's station, or point
> it at an address you don't already trust, without their explicit consent.

It can be used by humans. For the AI-agent  workflow, deployment steps, and the full TX
safety protocol, see [`.claude/skills/thetis-control/SKILL.md`](.claude/skills/thetis-control/SKILL.md).

## Install as an AI skill

This repo also ships a ready-to-use Claude Code skill
(`.claude/skills/thetis-control/`) that lets an AI agent operate a Thetis
radio through `thetisctl`, with the TX safety protocol built in. See
[`SKILL.md`](.claude/skills/thetis-control/SKILL.md) to install and deploy
it — it assumes `thetisctl` is already built and on `PATH` (below).

## Build

Prebuilt Linux/Windows/macOS (amd64+arm64) binaries are published to
[Releases](https://github.com/W5TSU/thetis-ai-skill/releases/latest) on every `v*` tag, alongside a `SHA256SUMS` file — or build from source (Go
1.22+, pure Go, no cgo, no dependencies):

```bash
go build -o thetisctl ./cmd/thetisctl
```

Either way, put the binary on `PATH` and confirm it:

```bash
chmod +x thetisctl-linux-amd64     # skip if built from source
sudo ln -sf "$(pwd)/thetisctl-linux-amd64" /usr/local/bin/thetisctl
thetisctl help
```

Windows steps (PowerShell symlink, or the Developer Mode caveat) are in
[`SKILL.md`'s "Getting the CLI"](.claude/skills/thetis-control/SKILL.md#1-deploying-this-skill).

## Enabling the servers in Thetis

Open **Setup** in the Thetis instance you want to control and turn on:

- **TCP/IP CAT Server** — for Tier 1 commands (`thetisctl cat ...`). Default
  bind `127.0.0.1:13013`.
- **TCI Server** — for Tier 2 commands (`thetisctl tci ...`). Default bind
  `127.0.0.1:50001`.

If `thetisctl` runs on a different machine than Thetis, rebind both to the
host's LAN IP (e.g. `192.168.1.50`) instead of `127.0.0.1` — see the
no-authentication warning above.

## Global flags

Both `cat` and `tci` take:

| Flag | Default | Meaning |
|---|---|---|
| `--host <ip>` | *(required)* | Thetis's address — never assumed local |
| `--port <n>` | `13013` (cat) / `50001` (tci) | Server port |
| `--timeout <duration>` | `3s` (cat) / `5s` (tci) | Network read/write timeout |

Every CAT and TCI command `thetisctl` currently implements — full tables,
one-liner effect, and example invocations for both tiers — is in
[`COMMANDS.md`](COMMANDS.md). For the full universe of commands each
protocol *defines* — including the ~470 CAT/TCI commands `thetisctl`
doesn't wire yet, and which ones are reachable via `tci query` in the
meantime — see [`PROTOCOLS.md`](PROTOCOLS.md).

## Transmitting (TX-capable commands)

These commands ()`cat ptt`, `cat tune`, `cat quickplay on`, `cat radae-sanity`, `tci tune`,
`tci ptt`, `tci cw send`, `tci cw send-msg`, and `tci tx-audio send`) can  ⚠️  key
your  transmitter. 

There is also `talk` (`talk/`, this repo's Python AI voice operator) — it runs in rehearsal mode by default (no TX), and transmits only when armed with `--armed --confirm-tx`; that flow is a bounded exception to the single-transmission model below, documented in  [`SKILL.md`'s §6](.claude/skills/thetis-control/SKILL.md) and `talk/README.md`. **Every one of the CLI commands above defaults to a dry  run** — without `--confirm-tx`, they print exactly what they would send and do nothing
TX-capable:

```
$ ./thetisctl tci --host 192.168.1.50 cw send 0 "CQ CQ DE W5TSU" --speed 5 --mode cwu
[dry-run] would send: modulation:0,cwu; cw_macros_speed:5; cw_macros:0,CQ CQ DE W5TSU;
Pass --confirm-tx=I-UNDERSTAND-THIS-KEYS-THE-RADIO to actually transmit this message.
```

To actually transmit, add the exact phrase (not a bare flag — this is deliberate, so nothing else can accidentally trigger it):

```bash
./thetisctl tci --host 192.168.1.50 cw send 0 "CQ CQ DE W5TSU" --speed 5 --mode cwu \
    --confirm-tx=I-UNDERSTAND-THIS-KEYS-THE-RADIO --max-duration 60s
```

Other TX flags:

| Flag | Applies to | Meaning |
|---|---|---|
| `--confirm-tx=I-UNDERSTAND-THIS-KEYS-THE-RADIO` | all TX-capable commands | Required to key for real; anything else stays a dry run |
| `--hold <duration>` (default `3s`, `15s` for `cat quickplay on`) | `cat ptt`, `cat quickplay on`, `tci tune`, `tci ptt` | Auto-unkeys/auto-stops after this long — `tci tune` caps this at 5s **total** (hold + confirm), regardless of what's requested; see below |
| `--max-duration <duration>` (default `10s` for tx-audio, `90s` for cw) | `tci tx-audio send`, `tci cw send` | Hard cap; truncates/stops and unkeys if exceeded |

Every TX-capable command unkeys automatically on completion, error, or
Ctrl-C, and **verifies** the unkey actually took effect (retrying if not)
rather than firing the command once and trusting it worked — a real prior
incident showed a bare fire-and-forget unkey can silently fail; see
[`NOTES.md`](NOTES.md#unkeying-is-confirmed-not-fire-and-forget). `tci tune`
additionally hard-caps total on-time at 5 seconds no matter what `--hold`
requests.

## Live tests

`go test ./...` only runs unit tests. Round-tripping every command against a real Thetis instance needs the `live` build tag and a few env vars — see
[`AGENTS.md`'s Verification section](AGENTS.md#verification) for the exact commands, which files they cover, and the separate opt-in required to actually key the transmitter (`txlive_test.go`) — not something an AI agent should ever run unprompted; see `SKILL.md`'s safety protocol.

## Known gotchas

A few headliners — the full incident history and root-cause writeups are in
[`NOTES.md`](NOTES.md):

- **Split routes TX to VFO B's frequency, not VFO A's.** Check before
  transmitting if split state is unknown.
- **`quickplay on` (and anything that reuses it, like `radae-sanity`) can
  key MOX for real**, not just inject RX I/Q — depends on a Thetis setting
  this tool can't read remotely.
- **Preamp attenuation is quantized** to a small set of discrete dB steps,
  not continuous.
- **CAT's `atten get` can hang** on a live, actively-receiving radio, while
  `atten set` never does.

## Extending thetisctl

Wire formats were confirmed by reading Thetis's own source, not just protocol docs — see [`SKILL.md`'s "Extending the command set"](.claude/skills/thetis-control/SKILL.md#extending-the-command-set) for the exact files and gotchas to check before adding new commands.

## talk — AI voice operator

`talk/` is a separate Python subsystem, not part of `thetisctl` itself: an AI that hears voice over the radio and answers back over the radio, on analog SSB/FM, while a human operator supervises. It drives `thetisctl` as a child process for every radio interaction — the Go tool above needs no changes to  upport it. Full design, glossary, and status live in [`talk/README.md`](talk/README.md); this section is the quick-start.

**Two modes, one flag.** Plain `python -m talk --config talk/config.toml` runs **rehearsal mode**: it listens continuously, transcribes locally (faster- hisper), recognizes when it's been addressed (a fuzzy match on your phonetic callsign or a configured wake word), composes a reply (canned rules first,  laude for anything else), synthesizes it (Piper),  and plays it on your local speakers. The radio is never keyed. Adding `--armed --confirm-tx I- NDERSTAND-THIS-KEYS-THE-RADIO` transmits for real instead — see [`SKILL.md`'s §6](.claude/skills/thetis-control/SKILL.md) for the safety model this depends on before ever using it. **Only a  uman operator, at their own terminal, should ever run the armed form** — never an agent. 

**Setup:**

```bash
cd talk
./setup.sh                              # venv, faster-whisper, Piper, model downloads
cp config.toml.example config.toml
$EDITOR config.toml                     # radio host, your callsign/phonetics, scripted texts
.venv/bin/python -m talk --config config.toml --check   # validate config, print the station banner
```

Prerequisites: both Thetis servers enabled and reachable — CAT (`13013`) for radio-state polling, TCI (`50001`) for audio and (when armed) transmit —  ame as the rest of this README's ["Enabling the servers in Thetis"](#enabling-the-servers-in-thetis). An `ANTHROPIC_API_KEY` in the environment enables the Claude half of the brain;  without it, `talk` runs rules-only and says so at startup.

**Tuning before ever arming:** run rehearsal mode against the real radio (RX-only, always safe) and watch how reliably it endpoints utterances and recognizes your callsign against real band noise before considering an armed session. `[vad]` in `config.toml` — `threshold_ratio`, `hangover_ms`, `min_utterance_ms` — is what to adjust: a `threshold_ratio` that's too low makes band noise trigger false utterances; too high and the leading syllable of  real speech gets clipped before onset confirms. The session log's JSONL records every utterance's duration, transcript, and trigger score — use that to see what's actually being detected before adjusting thresholds blind. `talk/tests/live_armed.md` is the human-only procedure for the first armed session, once rehearsal behaves well.
