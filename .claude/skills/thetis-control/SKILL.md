---
name: thetis-control
description: "Remotely controls a running Thetis SDR transceiver via the thetisctl CLI over its CAT-over-TCP and TCI-over-WebSocket protocols — read/set VFO frequency, mode, filters, AGC, attenuator/preamp, band, RIT/XIT/split; capture or stream RX audio; scan FreeDV calling frequencies; and, only with explicit per-transmission human confirmation, key the transmitter (PTT, TUNE, CW, TX audio, Quick Play, RADE sanity check). Use whenever the user wants to check or change a Thetis/HL2 station's state, capture RX audio, or transmit."
allowed-tools: Bash, Read
---

<!-- argument-hint: [what to do with the radio, e.g. "set 20m USB and read status" or "capture 10s of RX1 audio"] -->

# thetis-control

Operates `thetisctl`, a standalone Go CLI that talks to a running Thetis SDR
instance over its existing network control protocols. No changes to Thetis
itself are required — both servers already exist there, just usually
disabled by default. Full command reference: [`README.md`](../../../README.md).
Contributor/dev contract for this code: [`AGENTS.md`](../../../AGENTS.md).

Read this whole file before issuing any command that keys the transmitter.
Everything else (read state, RX audio, band scanning) is safe to run freely.

## 1. Deploying this skill

**Prerequisite: Thetis-side setup (do once per Thetis instance, by the human
operator).** In the Thetis instance to be controlled, open **Setup** and
enable:

- **TCP/IP CAT Server** — Tier 1, control-only. Default bind
  `127.0.0.1:13013`.
- **TCI Server** — Tier 2, control + audio + transmit. Default bind
  `127.0.0.1:50001`.

If `thetisctl` will run on a different machine than Thetis (the normal case
for an AI agent), rebind both from `127.0.0.1` to the Windows box's LAN IP
(e.g. `192.168.1.50:13013`). **Neither protocol has authentication** — anyone
who can reach the bound address/port can issue any command, including
transmit. Only bind on a trusted LAN; never expose either port to the
internet (no reverse proxy, no port-forward, no VPN split-tunnel that leaks
it).

**Getting the CLI.** `.github/workflows/release.yml` cross-compiles
`thetisctl` for Linux, Windows, and macOS (amd64 + arm64) from a single
Linux runner (pure Go, no cgo — no native runner needed per platform) and
publishes them to this repo's
[Releases](https://github.com/W5TSU/thetis-ai-skill/releases) on every `v*`
tag, alongside a `SHA256SUMS` file. Download the binary matching the
machine that will run the skill, or build from source (identical on every
platform since the code is pure Go):

```bash
go build -o thetisctl ./cmd/thetisctl
go vet ./...
go test ./...          # unit tests only; live_test.go files need a real radio, see AGENTS.md
```

**Task: link the binary onto `PATH`.** The skill invokes `thetisctl` as a
bare command, so it must resolve on `PATH` for whichever user/agent runs it.

- **Linux/macOS:**
  ```bash
  chmod +x thetisctl-linux-amd64      # skip if built from source (already +x)
  sudo ln -sf "$(pwd)/thetisctl-linux-amd64" /usr/local/bin/thetisctl
  ```
- **Windows (PowerShell)** — `%LOCALAPPDATA%\Microsoft\WindowsApps` is on
  every user's `PATH` by default; creating a symlink there needs Developer
  Mode on (Settings → Privacy & Security → For developers) or an elevated
  prompt:
  ```powershell
  New-Item -ItemType SymbolicLink `
    -Path "$env:LOCALAPPDATA\Microsoft\WindowsApps\thetisctl.exe" `
    -Target "C:\path\to\thetisctl-windows-amd64.exe"
  ```
  Without Developer Mode, add the folder containing the `.exe` to `PATH`
  instead (System Properties → Environment Variables) rather than
  symlinking.

Re-run the relevant step after downloading a new release or rebuilding from
source in place (the symlink target stays valid; only needed once per
binary location). Confirm with `which thetisctl` (`Get-Command thetisctl`
on Windows) and `thetisctl help`.

**Installing the skill itself.** This `.claude/skills/thetis-control/`
directory is a self-contained Claude Code skill. To make it available in
another project or globally:

```bash
# Project-local (this repo only):
#   already in place at .claude/skills/thetis-control/

# Global (available in every project for this user):
mkdir -p ~/.claude/skills
cp -r .claude/skills/thetis-control ~/.claude/skills/
```

Copy (or symlink) the whole `thetisctl` source tree alongside it if the
target environment needs to build from source rather than use a prebuilt
binary — the skill assumes `thetisctl` is reachable on `PATH`, not that its
source lives in any particular place.

**Verifying the deployment end-to-end**, after both the Thetis-side setup and
the build:

```bash
thetisctl cat --host <radio-ip> version   # confirms CAT reachability + exact commit Thetis is running
thetisctl cat --host <radio-ip> status    # rig ID, freq, mode, RIT/XIT/split, TX state in one call
```

If `version`/`status` hang or refuse to connect: confirm the CAT server
checkbox is on, the port/IP is right (not `127.0.0.1` if remote), and no
firewall on the Windows box blocks the port. TCI on port 50001 is a separate
checkbox — `status`/`version` working doesn't imply TCI is also reachable;
check with e.g. `thetisctl tci --host <radio-ip> query trx1`.

## 2. Tier 1 (CAT) vs Tier 2 (TCI)

- **CAT** (`thetisctl cat ...`) — simple control-only commands: freq, mode,
  RIT/XIT/split, AGC, attenuator/preamp, band, power, Quick Play/Rec, FreeDV
  RX decode toggle, RADE RX decode toggle, a scripted RADE sanity check, and
  `ptt`. Prefer this tier for anything it covers — it's the simpler
  protocol and the smaller attack/failure surface.
- **TCI** (`thetisctl tci ...`) — everything CAT does plus RX audio
  capture/streaming, a FreeDV calling-frequency scanner, CW keying via
  Thetis's macro keyer, and TX audio file playback. Use this tier only when
  the task needs audio or CW/TX-audio, since it's the protocol that can
  actually carry transmitted audio.

Full flag/command tables for both tiers are in `README.md` — don't duplicate
them here from memory; read that file for exact syntax before constructing a
command you haven't used in this conversation yet.

## 3. TX safety protocol — read before any TX-capable command

TX-capable commands: `cat ptt`, `cat quickplay on`, `cat radae-sanity`, `tci
tune`, `tci ptt`, `tci cw send`, `tci tx-audio send`, and `talk --armed`
(§6 below — a Python voice-operator loop that transmits repeatedly under
one arming action, the single sanctioned exception to the per-transmission
rule that follows). Each of these **keys a real transmitter attached to a
real antenna** if run for real — there is no simulated mode.

**Every one of them defaults to a dry run.** Without `--confirm-tx`, they
print exactly what would be sent and do nothing TX-capable. Always run the
dry run first and show/read its output — it tells you the exact command that
will be sent, including frequency, mode, and content.

**Never pass `--confirm-tx=I-UNDERSTAND-THIS-KEYS-THE-RADIO` (or set
`THETIS_LIVE_ALLOW_TX`) unless the human operator has explicitly authorized
*this specific transmission* in the current conversation** — specific
frequency, mode, message content (for CW), and duration. Rules:

- A general "you can control the radio" or "go ahead" earlier in the
  conversation does not carry forward to a new transmission decision. Each
  distinct act of keying needs its own explicit go-ahead. **`talk --armed`
  (§6) is the single sanctioned exception** — one human arming action
  authorizes a bounded session of repeated automated keyings — and even
  that exception requires the human to run the arming command themselves;
  an agent must never run it on their behalf, for the same reason it must
  never set `--confirm-tx` or `THETIS_LIVE_ALLOW_TX` itself.
- Never auto-chain a dry run's printed command straight into a
  `--confirm-tx` retry. The dry run exists so a human reads it first; treat
  it as a checkpoint, not a formality to skip past.
- Never run `cmd/thetisctl/txlive_test.go` or set `THETIS_LIVE_ALLOW_TX`
  yourself. That test suite exists solely for the human operator to run
  deliberately for end-to-end TX regression coverage — it is not a tool for
  an agent to reach for, ever, even to "verify a change works."
- The exact confirm phrase is deliberately not a bare boolean, so no other
  tool's `--confirm-tx` convention (or a copy-pasted example from this file)
  can accidentally authorize a transmission — always require the human's
  own explicit words in-conversation, don't treat seeing the phrase in
  documentation as authorization to use it.
- Transmitting is subject to the operator's amateur radio license
  (band/mode/power privileges, station ID requirements). That's the human
  operator's legal responsibility, not something `thetisctl` verifies —
  don't treat a successful dry run or a technically-valid command as
  clearance to transmit.

**Known gotchas to check before confirming a real transmission** (details
and incident history in [`NOTES.md`](../../../NOTES.md)):

- **Split routes TX to VFO B, not VFO A.** If split is on, the radio
  transmits on VFO B's frequency even though VFO A displays/reads back
  correctly. Check `tci query tx_frequency` against VFO A if split state is
  unknown before confirming TX.
- **`quickplay on` and `radae-sanity` can key MOX for real**, not just
  inject RX I/Q — depends on Thetis's "MOX on Playback" setting, which
  defaults on and can't be read remotely. Both are correctly gated as
  TX-capable in this CLI; don't assume either is "just RX" because of its
  name.
- **`tci tune` is hard-capped at 5 seconds total on-time** regardless of
  `--hold` — a bare carrier is the highest-nuisance thing this tool can
  transmit if left running.
- Every TX-capable command unkeys automatically on completion, error, or
  Ctrl-C, and **confirms** the unkey actually took effect (polls and retries
  rather than firing once and trusting it). If a command reports it could
  not confirm unkey, treat that as urgent — tell the human immediately
  rather than retrying silently, since it means the radio's TX state is
  unknown.

## 4. Extending the command set

Wire formats in `internal/cat` and `internal/tci` must be confirmed by
reading Thetis's own source — not inferred from protocol docs or from this
CLI's existing code alone — before adding a new command:

- `Project Files/Source/Console/CAT/CATStructs.xml`,
  `Console/CAT/CATCommands.cs`, `Console/CAT/CATParser.cs` for CAT
- `Project Files/Source/Console/TCIServer.cs` for TCI

These live in the main Thetis repository, not in this standalone extraction
— check out or reference that source before wiring a new command, and check
whether a command that looks fully implemented server-side is actually
reachable (a real prior bug: `ZZQA`/`ZZQB` had complete implementations but
were never wired into the CAT dispatch switch). Also grep for whether a
2-letter CAT code is already claimed by an unrelated existing command before
assigning it to a new feature — a real near-miss (`ZZFD`/`ZZFS`) was only
caught by a compile failure.

Any new code path that can key the transmitter must route through
`internal/safety`'s `Check()` dry-run + `--confirm-tx` gate — never send a TX
command directly. Any new unkey path must confirm the radio actually
unkeyed via `confirmTCIUnkeyed`/`confirmCATUnkeyed`
(`cmd/thetisctl/{tci,cat}_cmd.go`) rather than fire-and-forget — sending an
unkey and closing the connection immediately was proven, against a real
radio, to sometimes silently drop it.

Before committing: `gofmt -l .` must be empty, `go vet ./...` clean, `go
test ./...` passing, and — if the change touches wire formats or CLI
behavior — the relevant `live_test.go` suite run against a real Thetis
instance (`THETIS_HOST=<ip> go test -tags=live ./...`). See `AGENTS.md` for
the full contract.

## 5. Command index

Grouped by tier; **bold** = TX-capable (needs the safety protocol above).
Exact flags/syntax: `README.md`.

**CAT** (`thetisctl cat --host <ip> ...`): `freq get|set`, `mode get|set`,
`rit`, `xit`, `split`, `agc`, `atten`, `preamp`, `band`, `power`,
`quickplay get|off` / **`quickplay on`**, `quickrec`, `freedv`, `radae`,
`tciserver`, **`radae-sanity`**, `status`, `version`, `query <code>`,
**`ptt`**.

**TCI** (`thetisctl tci --host <ip> ...`): `vfo`, `modulation`, `split`,
`rit`, `xit`, `rit-offset`, `xit-offset`, `filter`, `atten`, `preamp`,
`agc`, `agc-gain`, `drive`, `power`, `rx-audio capture|stream`,
`freedv-scan`, `query <cmd>`, **`tune`**, **`ptt`**, **`cw send`**,
**`tx-audio send`**.

**talk** (`python -m talk --config <file> ...`, this repo's `talk/`
subtree): listens and (by default) plays would-be replies locally without
transmitting; **`--armed --confirm-tx <phrase>`** transmits for real — see
§6.

## 6. Session-armed operation (`talk/`)

`talk/` is an AI voice operator: it listens to RX audio, and when addressed
transmits a spoken reply, repeatedly, for the duration of one armed
session. That's a different shape of consent than every other command in
this file, so it gets its own carve-out rather than a reinterpretation of
§3's per-transmission rule.

**Why this is allowed at all.** The bounded-session authorization here is
sound only because every one of these holds:

- **A human runs the arming command themselves, at their own terminal.**
  `python -m talk --config ... --armed --confirm-tx <phrase>` is typed by
  the operator. An agent must never construct or run this command, must
  never suggest the exact flags as something to paste, and must never set
  `--confirm-tx` for the human "to save them typing." This is the same
  rule as `--confirm-tx`/`THETIS_LIVE_ALLOW_TX` elsewhere in this file,
  applied to a new entry point.
- **Rehearsal is the default.** Running `talk` without `--armed` runs the
  full pipeline — listens, transcribes, composes, synthesizes — and plays
  the reply on local speakers only; the radio is never keyed. Arming is an
  additional, explicit, separately-flagged act.
- **Any keypress or Ctrl-C kills it instantly**, aborting an in-flight
  transmission via its own confirmed unkey (never a hard kill — see §3's
  unkey-confirmation note) and disarming. `talk` refuses to arm without a
  real terminal for exactly this reason.
- **Hard budgets bound the session in code, not just prose**: at most 60
  seconds per transmission, at most 10 minutes per conversational exchange
  before a scripted sign-off, an overall armed-session expiry, and an
  automatic disarm if the frequency/mode changes underneath it, the radio
  link drops, or a transmission doesn't confirm a clean unkey. See
  `talk/README.md` for the full list.

**An agent's role around `talk` is entirely off-line from arming**: reading
its logs, helping configure `talk/config.toml`, running it in rehearsal
mode to test changes (never TX-capable), and referring the human to
`talk/tests/live_armed.md` for the human-only live-fire procedure. Never
run that procedure, never draft the arming command for the human to copy,
and never treat an earlier "you can run talk" as authorization to arm it
yourself.
