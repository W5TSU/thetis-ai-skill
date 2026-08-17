# thetisctl

A Go CLI for remotely controlling a running Thetis SDR instance over its
existing network protocols: CAT-over-TCP (frequency, mode, filters, VFO,
RIT/XIT, split, AGC, attenuator/preamp, band) and TCI-over-WebSocket (the
same controls plus RX/TX audio streaming, CW keying, and transmit). No
Thetis-side code changes are required — both servers already exist in
Thetis, they're just usually off by default.

This file is a plain command reference for a human running `thetisctl`
directly. For the AI-agent workflow, deployment steps, and the full TX
safety protocol, see
[`.claude/skills/thetis-control/SKILL.md`](.claude/skills/thetis-control/SKILL.md).

## Enabling the servers in Thetis

Open **Setup** in the Thetis instance you want to control and turn on:

- **TCP/IP CAT Server** — for Tier 1 commands (`thetisctl cat ...`). Default
  bind `127.0.0.1:13013`. If `thetisctl` runs on a different machine, rebind
  to the host's LAN IP (e.g. `192.168.1.50:13013`), not `127.0.0.1`.
- **TCI Server** — for Tier 2 commands (`thetisctl tci ...`). Default bind
  `127.0.0.1:50001`, same rebind note.

Neither server has authentication — anyone who can reach the bound
address/port can issue any command, including keying the transmitter. Only
bind these on a trusted LAN; never expose them to the internet.

## Build

```bash
go build -o thetisctl ./cmd/thetisctl
go vet ./...
go test ./...
```

Pure Go, no cgo, no external dependencies — builds anywhere Go runs.

## Global flags

Both `cat` and `tci` take:

| Flag | Default | Meaning |
|---|---|---|
| `--host <ip>` | *(required)* | Thetis's address — never assumed local |
| `--port <n>` | `13013` (cat) / `50001` (tci) | Server port |
| `--timeout <duration>` | `3s` (cat) / `5s` (tci) | Network read/write timeout |

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
| `quickplay on` | **TX-capable** — see [Transmitting](#transmitting-tx-capable-commands); injects `Music\Thetis\quickrecord\SDRQuickAudio.wav` as RX I/Q ahead of the antenna, which is how FreeDV decode tests can be triggered remotely, but the underlying Thetis function can also key MOX for real — see [Notes](#notes-on-real-world-behavior) |
| `quickrec get` | Read whether Quick Rec is active |
| `quickrec on\|off` | Quick Rec: record RX audio to that same fixed file |
| `freedv get` | Read whether FreeDV RX decode is enabled |
| `tciserver get` | Read whether Thetis's TCI server is listening |
| `tciserver on\|off` | Enable/disable the TCI server — works even when TCI itself is unreachable (CAT doesn't depend on it being up), so it can bootstrap TCI back on after a restart left the Setup checkbox unchecked |
| `freedv on\|off` | Enable/disable FreeDV RX decode (`fdv.c`), RX1 only |
| `freedv status` | Read FreeDV sync + SNR — read-only, e.g. `SYNC  SNR 15.3 dB` or `no sync` |
| `status` | Rig ID + frequency/mode/RIT/XIT/split/TX in one call |
| `version` | Software version string, incl. the git short SHA the running build was made from (`ZZZV`) — verifies which commit a remote instance is actually running |
| `query <CODE>` | Raw passthrough for any CAT command without a dedicated wrapper (e.g. `query ZZZV`) |
| `ptt on\|off` | **TX-capable** — see [Transmitting](#transmitting-tx-capable-commands) |

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
| `power on\|off` | Start/stop Thetis's radio engine, **not mains power**; waits for the server's confirmation broadcast |
| `rx-audio capture <rx> --duration <d> --out <file.wav>` | Record RX audio to a WAV file |
| `rx-audio stream <rx> --duration <d>` | Stream RX audio as raw float32 LE PCM to stdout |
| `freedv-scan [--dwell 6s] [--out-dir <dir>]` | RX-only — tunes RX1 through the FreeDV calling frequencies, records a WAV per band, reports peak/RMS, restores original freq/mode when done |
| `tune <rx> on\|off` | **TX-capable** — key TUNE (bare carrier); hard-capped at 5s total on-time |
| `ptt <rx> on\|off [--audio]` | **TX-capable** — key PTT (`--audio` marks this connection as the TX audio source) |
| `cw send <rx> "<text>" --speed <wpm> --mode <cw\|cwu\|cwl>` | **TX-capable** — key CW text via Thetis's own macro keyer |
| `tx-audio send <rx> --file <wav>` | **TX-capable** — stream a WAV file as TX audio |
| `query <cmd> [args...]` | Raw passthrough — send any TCI text command not listed above and print the reply |

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

## FreeDV Reporter spotting (`thetisctl freedv-reporter watch`)

Watches [FreeDV Reporter](https://qso.freedv.org)'s live activity feed for stations
starting to transmit within a frequency range, and — with `--tci` — automatically
retunes Thetis's RX1 there. **Not TX-capable**: it only changes what Thetis is
listening to, never keys anything, so it's safe to leave running unattended.

| Flag | Effect |
|---|---|
| `--min-freq <hz>` / `--max-freq <hz>` | Frequency range to watch (default: 20m, 14000000–14350000) |
| `--tci <ip>` [`--tci-port 50001`] | Auto-tune this Thetis instance's RX1 on activity |
| `--mode <mode>` | Mode to set when auto-tuning (default `digu`) |

```bash
./thetisctl freedv-reporter watch                      # just print alerts, don't tune anything
./thetisctl freedv-reporter watch --tci 192.168.1.50    # also auto-tune that Thetis's RX1
```

The reporter has no REST/JSON API — its live data is a Socket.IO v4 feed, confirmed
by direct protocol probing (`internal/freedvreporter`, a hand-rolled client matching
`internal/tci`'s existing no-third-party-dependency convention). Runs until Ctrl-C.

## Transmitting (TX-capable commands)

`cat ptt`, `cat quickplay on`, `tci tune`, `tci ptt`, `tci cw send`, and `tci
tx-audio send` can key the transmitter. **Every one of them defaults to a dry
run** — without `--confirm-tx`, they print exactly what they would send and
do nothing TX-capable:

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
Ctrl-C, and — as of a fix made after a real incident, see below —
**verifies** the unkey actually took effect (retrying if not) rather than
firing the command once and trusting it worked.

**Unkeying is confirmed, not fire-and-forget.** Earlier versions of every
TX-capable command sent their unkey and closed the connection immediately
afterward; against a real radio this was shown to sometimes silently drop
the command, leaving the radio keyed with no time bound until a human
noticed. Every unkey now sends, then queries the radio's actual state and
retries until confirmed (or reports an error — never silent success). `tci
tune` additionally hard-caps total on-time at 5 seconds no matter what
`--hold` requests, since a bare unmodulated carrier is the highest-nuisance
thing this tool can transmit if left running.

## Live tests

Four files, all build-tag `live` (excluded from `go test ./...` and CI):

| File | Covers |
|---|---|
| `internal/cat/live_test.go` | Every exported CAT client function |
| `internal/tci/live_test.go` | Every exported TCI client function |
| `cmd/thetisctl/live_test.go` | CLI-layer code the library tests bypass: `rx-audio capture`/`stream`, `query`, and a dry run of every TX-capable command |
| `cmd/thetisctl/txlive_test.go` | **Opt-in only** — actually keys the transmitter for real; see below |

```bash
THETIS_HOST=192.168.2.12 go test -tags=live ./internal/cat/... -v
THETIS_HOST=192.168.2.12 go test -tags=live ./internal/tci/... -v
THETIS_HOST=192.168.2.12 go test -tags=live ./cmd/thetisctl/... -v
```

Every settable function is round-tripped (read → change → verify → restore
original) rather than asserting a fixed value. The first three files never
transmit for real — TX-capable functions/commands are only ever exercised in
their safe form (`SetPTT`/`SetTrx`/`SetTune` called with `false`; CLI
dry-runs with no `--confirm-tx`). See the test file doc comments for
exceptions (e.g. `SetBand` is read-only in the test; preamp attenuation is
quantized, not continuous — see below).

**`txlive_test.go` actually transmits.** It requires a *second*, independent
env var beyond `THETIS_HOST`, set to the exact confirm phrase, or every test
in it skips:

```bash
THETIS_HOST=192.168.2.12 THETIS_LIVE_ALLOW_TX=I-UNDERSTAND-THIS-KEYS-THE-RADIO \
    go test -tags=live ./cmd/thetisctl/... -run TestLiveTX -v
```

Run this yourself, deliberately, when you want real end-to-end TX
regression coverage — it's not something an AI agent should ever run
unprompted; see `SKILL.md`'s safety protocol.

## Notes on real-world behavior

- **`version`'s build date field can show garbled text on some machines —
  a pre-existing cosmetic bug, not something `version`/`query` introduced.**
  `VersionInfo.BuildDate` is generated by a `wmic os get localdatetime`
  parse in `Thetis.csproj`'s pre-build event, which assumes a fixed locale
  date format; on at least one real Windows instance this produced
  `(~4,2/~6,2/~2,2)` instead of a real date. The **git SHA** portion
  (`git:<sha>`, added 2026-08-07 specifically so a remote build's exact
  commit can be verified) is unaffected — it comes from a separate
  `git rev-parse --short HEAD` call in the same pre-build step.
- **`quickplay on` can key MOX for real — discovered by live testing
  2026-08-04, previously undocumented and previously ungated.** Quick Play
  was designed and documented (including earlier in this file) as an
  RX-only bench-test feature: it injects a wav as RX I/Q ahead of the
  antenna input, bypassing the antenna entirely. In practice it calls
  Thetis's `PlayFileViaWDSP` (`Console/clsAudioRecordPlayback.cs`), a
  function *shared* with a genuine TX-audio-preview feature, which contains
  `if (!_console.MOX && MoxOnPlayback) _console.MOX = true;` —
  and `MoxOnPlayback` **defaults to `true`** in this codebase, set via
  Setup → Recording's "MOX on Playback" checkbox. Before this was caught,
  `quickplay on` went through `catToggle` exactly like `quickrec` or
  `freedv`, with no TX gate at all — every call could have kept MOX on,
  completely bypassing `--confirm-tx`. `quickplay on` is now treated as
  TX-capable (see [Transmitting](#transmitting-tx-capable-commands));
  `quickrec` was checked and confirmed to have no equivalent MOX side
  effect (`RecordToFileFromWDSP` never touches `_console.MOX`), so it
  stays ungated. If you need Quick Play to be genuinely RX-only, confirm
  "MOX on Playback" is unchecked in Thetis's Setup → Recording tab — this
  tool cannot read or change that setting remotely, only the resulting MOX
  state.
- **Split routes TX to VFO B's frequency, not VFO A's — check it before
  transmitting.** If split is enabled, everything you've set on VFO A
  (frequency, mode) stays displayed correctly, but the radio actually
  transmits on VFO B's frequency instead. This caused a real incident: a
  test session transmitted on a different band than intended because split
  had been on since before the session started, and nothing in routine
  `status` output made that obvious. Check TCI's `query tx_frequency`
  against VFO A before transmitting if split state is unknown.
- **CAT connect banner.** If Thetis's "Send Welcome" option is on, connecting
  to the CAT port gets you an unsolicited `#Thetis TCP/IP Cat - <version>#;`
  line before any command reply. `thetisctl` already accounts for this.
- **CW completion isn't a "message finished" event.** Thetis's TCI protocol
  has no such event for a plain `cw_macros` send (`cw_macros_empty` only
  fires in CW Terminal mode, which `cw send` doesn't use). `cw send` instead
  polls live PTT state and reports done once it sees the radio key up and
  then release.
- **Switching to CW mode** can shift the displayed/tuned frequency by
  Thetis's CW pitch offset (commonly 600 Hz) — this is normal sidetone-offset
  behavior, not a bug in `thetisctl`.
- **The classic Kenwood `PS` (power) CAT command is a disabled stub** — use
  `power` (which wraps the real, active `ZZPS`/TCI `start;`/`stop;`
  commands) instead.
- **A fully-implemented CAT command can still be unreachable.** `quickplay`/
  `quickrec`'s underlying `ZZQA`/`ZZQB` had complete, correct
  implementations in Thetis's `CATCommands.cs` that were simply never
  wired into the dispatch switch or given a `CATStructs.xml` entry — no CAT
  client, this tool included, could reach them until that was fixed
  (2026-07-30). Worth remembering if a command that looks fully implemented
  in source still doesn't respond over the wire.
- **`freedv on|off|status` (`ZZDV`/`ZZDS`) is new, not revived** — added
  2026-07-30 specifically to make FreeDV RX decode testing (`fdv.c`, still
  under active development on the FreeDV branch) scriptable without a human
  watching the Setup DSP tab: `freedv on`, `quickplay on --confirm-tx=...`
  to inject the bench test signal (see the MOX-on-playback note above —
  this is TX-capable), then `freedv status` for an objective sync/SNR
  readout.
  `ZZFD`/`ZZFS` were already taken by unrelated existing commands (FM
  deviation, RX2 filter low) — a real near-miss caught only because CI
  failed to compile (`CS0111: already defines a member`), a good reminder
  to grep for an unused code before claiming one, the same way `quickplay`/
  `quickrec`'s revival required checking `ZZQA`/`ZZQB` weren't already live.
- **TCI's initial-state burst can shadow a reply right after connect.** If
  "send initial state on connect" is on, Thetis pushes ~100+ unsolicited
  status frames (ending in a `ready;` sentinel) immediately after the
  WebSocket handshake. A request issued too early can match a stale value
  from that burst instead of the genuine reply — this was a real bug in `tci
  query`'s raw passthrough (fixed: it now matches replies by command name
  instead of taking whichever frame arrives first; see `tciQuery`'s doc
  comment for the residual ambiguity that fix can't fully resolve) and was
  also hit by the live test suite's own query helper (see
  `drainInitialBurst` in `internal/tci/live_test.go`).
- **Preamp attenuation is quantized, not continuous.** Both CAT's `ZZPA` and
  TCI's `rx_preamp_att_ex` resolve to a small set of discrete steps (0, -10,
  -20, -30, -40, -50 dB, plus SA-prefixed variants) server-side — an
  in-between value gets silently snapped to the nearest step.
- **CAT's `atten get` (`ZZRX` query) can hang indefinitely on a live,
  actively-receiving radio — but `atten set` doesn't.** Independent of
  `power` state; confirmed `SetAttenuatorDB` always returns instantly (it's
  fire-and-forget over CAT and never waits for a reply) while `GetAttenuatorDB`
  hangs. Suspected cause: an automatic overload-protection feature was
  observed changing the equivalent TCI value (`rx_step_att_ex`) on its own in
  real time, possibly saturating the UI-thread queue the CAT getter blocks
  on. Unconfirmed — if you hit this, cross-check via TCI (`tci atten <rx>
  ...`) before assuming `thetisctl` is at fault.

## Extending thetisctl

Wire formats were confirmed by reading Thetis's own source, not just
protocol docs — see
[`SKILL.md`'s "Extending the command set"](.claude/skills/thetis-control/SKILL.md#extending-the-command-set)
for the exact files and gotchas to check before adding new commands.
