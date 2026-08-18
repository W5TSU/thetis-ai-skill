# thetisctl

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go CLI for remotely controlling a running Thetis SDR instance over its
existing network protocols: CAT-over-TCP (frequency, mode, filters, VFO,
RIT/XIT, split, AGC, attenuator/preamp, band) and TCI-over-WebSocket (the
same controls plus RX/TX audio streaming, CW keying, and transmit). No
Thetis-side code changes are required — both servers already exist in
Thetis, they're just usually off by default.

> **⚠️ Only point this at a radio you directly control.** `thetisctl` can key
> a real transmitter into a real antenna — its TX-capable commands cause
> actual on-air RF, not a simulation. Only use it against a Thetis instance
> you personally own or operate, are licensed to transmit from, and can
> physically reach or shut down if something goes wrong. Neither of Thetis's
> control protocols has authentication, so anything reachable on the network
> is controllable — never run this against someone else's station, or point
> it at an address you don't already trust, without their explicit consent.

This file is a plain command reference for a human running `thetisctl`
directly. For the AI-agent workflow, deployment steps, and the full TX
safety protocol, see
[`.claude/skills/thetis-control/SKILL.md`](.claude/skills/thetis-control/SKILL.md).

## Install as an AI skill

This repo also ships a ready-to-use Claude Code skill
(`.claude/skills/thetis-control/`) that lets an AI agent operate a Thetis
radio through `thetisctl`, with the TX safety protocol built in. See
[`SKILL.md`](.claude/skills/thetis-control/SKILL.md) to install and deploy
it — it assumes `thetisctl` is already built and on `PATH` (below).

## Build

Prebuilt Linux/Windows/macOS (amd64+arm64) binaries publish to
[Releases](https://github.com/W5TSU/thetis-ai-skill/releases/latest) on
every `v*` tag, alongside a `SHA256SUMS` file — or build from source (Go
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

The two tables below cover every command `thetisctl` currently implements.
For the full universe of commands each protocol *defines* — including the
~470 CAT/TCI commands `thetisctl` doesn't wire yet, and which ones are
reachable via `tci query` in the meantime — see
[`PROTOCOLS.md`](PROTOCOLS.md).

## CAT commands — control only (`thetisctl cat ...`)

| Command | Effect |
|---|---|
| `freq get A\|B` | Read VFO A or B frequency (Hz) |
| `freq set A\|B <hz>` | Set VFO A or B frequency, then reads it back to confirm |
| `mode get` | Read the demod mode |
| `mode set <name>` | Set mode: `USB`, `LSB`, `CW`, `CWL`, `FM`, `AM`, `DIGU`, `DIGL` |
| `rit on\|off\|get` | RIT enable/disable/query |
| `xit on\|off\|get` | XIT enable/disable/query |
| `split on\|off\|get` | VFO split enable/disable/query |
| `agc get` | Read AGC mode |
| `agc set <name>` | Set AGC: `FIXED`, `LONG`, `SLOW`, `MEDIUM`, `FAST`, `CUSTOM` |
| `atten get` | Read RX1 step attenuator (dB) |
| `atten set <0-31>` | Set RX1 step attenuator (dB) |
| `preamp set <0-9>` | Set RX1 preamp level (0=off, 1=on, 2-6=-10..-50dB, 7-9=SA -10..-30dB) |
| `band get` | Read current band |
| `band set <name>` | Set band: `160`,`80`,`60`,`40`,`30`,`20`,`17`,`15`,`12`,`10`,`6`,`2`,`GEN`,`WWV`,`V0`-`V13` |
| `power get` | Read whether Thetis's radio engine is running |
| `power on\|off` | Start/stop Thetis's radio engine — the main Power button, **not mains power** to the HL2 board |
| `quickplay get` | Read whether Quick Play is active |
| `quickplay off` | Stop Quick Play (always safe, no confirmation needed) |
| `quickplay on` | **TX-capable** — see [Transmitting](#transmitting-tx-capable-commands); injects `Music\Thetis\quickrecord\SDRQuickAudio.wav` as RX I/Q ahead of the antenna, which is how FreeDV/RADE decode tests can be triggered remotely, but the underlying Thetis function can also key MOX for real — see [`NOTES.md`](NOTES.md) |
| `quickrec get` | Read whether Quick Rec is active |
| `quickrec on\|off` | Quick Rec: record RX audio to that same fixed file |
| `freedv get\|on\|off` | Enable/disable/read FreeDV RX decode (`fdv.c`), RX1 only |
| `freedv status` | Read FreeDV sync + SNR — read-only, e.g. `SYNC  SNR 15.3 dB` or `no sync` |
| `radae get\|on\|off` | Enable/disable/read RADE V1 RX decode, RX1 only |
| `radae status` | Read RADE sync + SNR — read-only, same format as `freedv status` |
| `radae-sanity` | **TX-capable** — see [Transmitting](#transmitting-tx-capable-commands); scripts a full RADE off-air sanity check (tune, enable decode, inject a bench test signal via Quick Play, poll sync/SNR, clean up) and prints a summary — see [`NOTES.md`](NOTES.md) |
| `tciserver get` | Read whether Thetis's TCI server is listening |
| `tciserver on\|off` | Enable/disable the TCI server — works even when TCI itself is unreachable (CAT doesn't depend on it being up), so it can bootstrap TCI back on after a restart left the Setup checkbox unchecked |
| `status` | Rig ID + frequency/mode/RIT/XIT/split/TX in one call |
| `version` | Software version string, incl. the git short SHA the running build was made from (`ZZZV`) — verifies which commit a remote instance is actually running |
| `query <CODE>` | Raw read passthrough for any CAT command without a dedicated wrapper (e.g. `query ZZZV`) |
| `set <CODE> [params...]` | Raw write passthrough — refuses `TX`/`RX`/`ZZTX`/`ZZTU` (use `ptt`/`tune`, which apply the TX safety gate) |
| `zz list [prefix]` | List every registered `ZZxx` extended CAT command (~200), each classified bool/unsigned/signed/action, with valid range where known |
| `zz get <CODE>` / `zz set <CODE> <value>` | Validated get/set for any registered `ZZxx` command — the general-purpose way to reach the ~200 extended commands without a dedicated named wrapper; full list and methodology in [`PROTOCOLS.md`](PROTOCOLS.md) |
| `ptt on\|off` | **TX-capable** — see [Transmitting](#transmitting-tx-capable-commands) |
| `tune on\|off` | **TX-capable** — bare carrier via the Thetis-extended `ZZTU`; hard-capped at 5s total on-time, same as `tci tune` |

```bash
./thetisctl cat --host 192.168.1.50 freq set A 14074000
./thetisctl cat --host 192.168.1.50 mode set USB
./thetisctl cat --host 192.168.1.50 status
```

## TCI commands — control, audio, and transmit (`thetisctl tci ...`)

`rx` selects the receiver: `0` = RX1, `1` = RX2.

| Command | Effect |
|---|---|
| `vfo <rx> <chan 0\|1> <hz>` | Set VFO A(0)/B(1) frequency |
| `modulation <rx> <mode>` | Set mode: `lsb`,`usb`,`dsb`,`am`,`sam`,`fm`,`cw`,`cwl`,`cwu`,`digl`,`digu` |
| `split <rx> on\|off` | VFO split |
| `rit <rx> on\|off` | RIT enable |
| `xit <rx> on\|off` | XIT enable |
| `rit-offset <rx> <hz>` | RIT offset |
| `xit-offset <rx> <hz>` | XIT offset |
| `filter <rx> <lowHz> <highHz>` | RX filter passband edges |
| `atten <rx> <dB>` | Step attenuator (dB, ≥0) |
| `preamp <rx> <dB>` | Preamp gain, expressed as attenuation ≤0 (e.g. `-10`) |
| `agc <rx> <mode>` | AGC mode: `off`/`fixed`, `long`, `slow`, `medium`/`normal`, `fast`, `custom` |
| `agc-gain <rx> <-20..120>` | AGC gain/threshold |
| `drive <rx> <0-100>` | TX drive power |
| `tune-drive <rx> <0-100>` | TX drive power used while in TUNE mode |
| `power on\|off` | Start/stop Thetis's radio engine, **not mains power**; waits for the server's confirmation broadcast |
| `dds <rx> <hz>` | Retune the panorama/DDS center — moves the VFO along with it to preserve IF offset |
| `if-offset <rx> <chan 0\|1> <offsetHz>` | Set a VFO's frequency as an offset from the DDS center (the spec-native alternative to `vfo`) |
| `rx-enable <rx> on\|off` | Enable/disable RX2 as a software receiver (RX1 is always enabled) |
| `rx-channel <rx> <chan 0\|1> on\|off` | Enable/disable an additional receive channel (sub-receiver) |
| `nb\|bin\|nr\|anf\|apf\|nf <rx> on\|off` | DSP toggles: noise blanker, pseudo-stereo, noise reduction, auto-notch, CW audio-peak filter, tracking notch filter |
| `sql <rx> on\|off` | Squelch enable |
| `sql-level <rx> <-140..0 dB>` | Squelch threshold |
| `volume <-60..0 dB>` | Main RX volume |
| `mute on\|off` | Mute/unmute overall RX audio (both receivers) |
| `rx-mute <rx> on\|off` | Mute/unmute a single receiver |
| `rx-volume <rx> <chan 0\|1> <-60..0 dB>` | Per-channel RX audio gain |
| `rx-balance <rx> <chan 0\|1> <-40..40>` | Per-channel stereo balance |
| `rx-sensors on\|off [--interval <ms>]` | Enable/disable this connection receiving periodic S-meter broadcasts |
| `tx-sensors on\|off [--interval <ms>]` | TX-side counterpart to `rx-sensors` |
| `spot send <call> <mode> <hz> <argb> [text]` / `spot delete <call>` / `spot clear` | Push/remove/clear panadapter spots |
| `focus` | Bring Thetis's main window to the foreground |
| `iq-samplerate <hz>` | Negotiate IQ stream sample rate — cosmetic on current Thetis (echoed back, doesn't retune hardware) |
| `audio-samplerate <hz>` | Set RX audio stream sample rate: `8000`\|`12000`\|`24000`\|`48000` |
| `rx-audio capture <rx> --duration <d> --out <file.wav>` | Record RX audio to a WAV file |
| `rx-audio stream <rx> --duration <d>` | Stream RX audio as raw float32 LE PCM to stdout |
| `iq capture <rx> --duration <d> --out <file.wav>` | Record RX I/Q to a WAV file (always float32 — Thetis hard-codes IQ encoding) |
| `iq stream <rx> --duration <d>` | Stream RX I/Q as raw float32 LE PCM to stdout |
| `freedv-scan [--dwell 6s] [--out-dir <dir>]` | RX-only — tunes RX1 through the FreeDV calling frequencies, records a WAV per band, reports peak/RMS, restores original freq/mode when done |
| `tune <rx> on\|off` | **TX-capable** — key TUNE (bare carrier); hard-capped at 5s total on-time |
| `ptt <rx> on\|off [--audio]` | **TX-capable** — key PTT (`--audio` marks this connection as the TX audio source) |
| `cw send <rx> "<text>" --speed <wpm> --mode <cw\|cwu\|cwl>` | **TX-capable** — key CW text via Thetis's own macro keyer |
| `cw send-msg <rx> <prefix\|_> <call> <suffix> --speed <wpm> --mode <m>` | **TX-capable** — key CW with callsign-repeat/mid-transmission-edit support |
| `cw edit-callsign <call>` | Edit the callsign of an in-progress `cw send-msg` transmission (not TX-capable on its own) |
| `cw speed-up <wpm>` / `cw speed-down <wpm>` | Nudge the CW macro/message keyer speed |
| `cw delay <ms>` | Delay between keying TX and the CW engine starting to send |
| `cw terminal <rx> on\|off` | Toggle CW Terminal mode (stay keyed after a message finishes instead of auto-unkeying) |
| `tx-audio send <rx> --file <wav>` | **TX-capable** — stream a WAV file as TX audio |
| `query <cmd> [args...]` | Raw passthrough — send any TCI text command not listed above and print the reply |

Full per-command detail (argument ranges, wire format, what's cosmetic vs.
real on current Thetis) is in [`PROTOCOLS.md`](PROTOCOLS.md).

```bash
./thetisctl tci --host 192.168.1.50 vfo 0 0 14074000
./thetisctl tci --host 192.168.1.50 modulation 0 usb
./thetisctl tci --host 192.168.1.50 rx-audio capture 0 --duration 10s --out capture.wav
./thetisctl tci --host 192.168.1.50 rx-audio stream 0 --duration 10s > raw.pcm
./thetisctl tci --host 192.168.1.50 freedv-scan --out-dir /tmp/freedv-scan
```

`freedv-scan`'s peak/RMS numbers are a prioritization hint for which
captures to listen to, not FreeDV identification — spectral shape alone
can't reliably tell a real digital-voice signal apart from a mistuned SSB
voice transmission or plain band noise (confirmed the hard way: a capture
that looked "flat, broadband, no speech pauses" turned out to just be
mistuned voice). Listen to the files yourself, or run them through actual
FreeDV software, to confirm.

## Transmitting (TX-capable commands)

`cat ptt`, `cat tune`, `cat quickplay on`, `cat radae-sanity`, `tci tune`,
`tci ptt`, `tci cw send`, `tci cw send-msg`, and `tci tx-audio send` can key
the transmitter. **Every one of them defaults to a dry run** — without
`--confirm-tx`, they print exactly what they would send and do nothing
TX-capable:

```
$ ./thetisctl tci --host 192.168.1.50 cw send 0 "CQ CQ DE W5TSU" --speed 5 --mode cwu
[dry-run] would send: modulation:0,cwu; cw_macros_speed:5; cw_macros:0,CQ CQ DE W5TSU;
Pass --confirm-tx=I-UNDERSTAND-THIS-KEYS-THE-RADIO to actually transmit this message.
```

To actually transmit, add the exact phrase (not a bare flag — this is
deliberate, so nothing else can accidentally trigger it):

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

`go test ./...` only runs unit tests. Round-tripping every command against a
real Thetis instance needs the `live` build tag and a few env vars — see
[`AGENTS.md`'s Verification section](AGENTS.md#verification) for the exact
commands, which files they cover, and the separate opt-in required to
actually key the transmitter (`txlive_test.go`) — not something an AI agent
should ever run unprompted; see `SKILL.md`'s safety protocol.

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

Wire formats were confirmed by reading Thetis's own source, not just
protocol docs — see
[`SKILL.md`'s "Extending the command set"](.claude/skills/thetis-control/SKILL.md#extending-the-command-set)
for the exact files and gotchas to check before adding new commands.
