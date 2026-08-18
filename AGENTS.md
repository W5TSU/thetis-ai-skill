# thetisctl (Thetis AI-control CLI)

## Purpose

`thetisctl` is a standalone Go CLI that gives an AI agent (or any script)
remote control of a running Thetis instance over its existing CAT-over-TCP
and TCI-over-WebSocket network protocols — control state, RX/TX audio, and
transmit. Companion to `.claude/skills/thetis-control/SKILL.md`, which is the
canonical usage and safety-protocol document; keep this file and that skill
consistent when either changes.

## Ownership

This repo is a standalone extraction of what was originally the
`Tools/thetis-ai-control/` subtree of the main Thetis repo. It is not part
of the VS solution, not built by that repo's `.github/workflows/build.yml`
(the Windows Thetis build), and has no dependency on `Project Files/` at
build/run time — only at dev time, when confirming a new wire format against
Thetis's own source (see "Extending the command set" in
`.claude/skills/thetis-control/SKILL.md`).

## Local Contracts

- `gofmt -l .` must be empty; `go vet ./...` must be clean.
- No cgo, no live local audio device I/O (mic/speaker) — audio is file
  (WAV) and stdin/stdout PCM only, so the tool stays pure Go and builds
  anywhere. This is a deliberate scope decision, not a placeholder for later.
- Any code path that can key the transmitter (CAT `TX;`, TCI `trx:...,true`,
  TCI `tune:...,true`) must route through `internal/safety`'s dry-run +
  `--confirm-tx` literal-match gate — never key directly.
- Every unkey (`tune off`, `ptt off`, auto-unkey-after-`--hold`, completion/
  interrupt/error unkey in `tx-audio`/`cw send`) must confirm the radio
  actually unkeyed before returning — via `confirmTCIUnkeyed`/
  `confirmCATUnkeyed` in `cmd/thetisctl/{tci,cat}_cmd.go` — never a bare
  fire-and-forget send. Sending a TX-off command and immediately closing the
  connection was proven, against a real radio, to sometimes silently drop
  it, leaving the transmitter keyed with no time bound; see
  `.claude/skills/thetis-control/SKILL.md`'s gotchas for the incident.
- Wire formats in `internal/cat` and `internal/tci` were confirmed by reading
  `Project Files/Source/Console/CAT/{CATStructs.xml,CATCommands.cs,CATParser.cs}`
  and `Project Files/Source/Console/TCIServer.cs` directly (comments on each
  typed helper cite line numbers). Re-verify against those files — not just
  each other — before changing a wire format, since Thetis is periodically
  synced from upstream and command dispatch can move.

## Work Guidance

- Dry-run-by-default with an explicit, hard-to-fat-finger `--confirm-tx`
  phrase for anything TX-capable is a durable, user-stated safety
  requirement (neither Thetis network protocol has authentication, and TCI
  TX audio genuinely transmits RF) — do not relax it for convenience.

## Verification

```bash
gofmt -l .            # must be empty
go vet ./...
go test ./...
go build -o thetisctl ./cmd/thetisctl
```

In the original monorepo, `.github/workflows/thetisctl.yml` ran the above on
push/PR touching this directory, independent of the Windows-only
`build.yml`. This standalone extraction has no CI configured yet — add a
workflow here if this repo gets its own remote.

`internal/cat/live_test.go`, `internal/tci/live_test.go`, and
`cmd/thetisctl/live_test.go` (build tag `live`, excluded from the above)
round-trip every exported client function and every non-TX CLI code path
against a real, running Thetis instance — the primary regression check for
this package now that it exists, run it after any wire-format or CLI change:

```bash
THETIS_HOST=<radio-ip> go test -tags=live ./internal/cat/... -v
THETIS_HOST=<radio-ip> go test -tags=live ./internal/tci/... -v
THETIS_HOST=<radio-ip> go test -tags=live ./cmd/thetisctl/... -v
```

None of these three ever exercise TX-capable functions for real (see the
test files' doc comments — TX-capable calls/commands are only ever made in
their safe `false`/dry-run form). `cmd/thetisctl/txlive_test.go` is a
separate, fourth file that actually keys the transmitter; it requires a
second env var (`THETIS_LIVE_ALLOW_TX`, exact match against
`internal/safety.ConfirmPhrase`) beyond `THETIS_HOST` or every test in it
skips. It exists solely for a human operator to run deliberately for
end-to-end TX regression coverage — an agent must never run it, or set that
env var, without the same explicit, per-invocation, in-conversation
go-ahead required for `--confirm-tx` itself; see
`.claude/skills/thetis-control/SKILL.md`'s safety protocol.

## Child DOX Index

- `PROTOCOLS.md` — exhaustive TCI/CAT command inventory generated from the
  upstream TCI spec and Thetis's own CAT source, with an implemented-vs-not
  breakdown; regenerate it (steps included) after a Thetis sync that touches
  CAT/TCI dispatch.
- `NOTES.md` — real-world behavior notes and incident history (split out of
  `README.md` to keep that a plain command reference); add to it, don't
  duplicate back into `README.md`, when a new gotcha is confirmed against a
  real radio.
