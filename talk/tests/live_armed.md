# Live armed test procedure (human operator only)

This is the one step in the `talk` feature that an AI agent must never run.
Read [`SKILL.md`'s §6](../../.claude/skills/thetis-control/SKILL.md) first
for why the session-armed carve-out exists and what bounds it.

Everything up to this point — rehearsal runs, config tuning, the offline
test suite, a rehearsal soak against the real radio — is safe to run freely
and repeatedly, by a human or an agent. This procedure is different: it
ends with the station transmitting for real.

## Before you start

- [ ] **Dummy load.** Point the transmitter at a dummy load, not an
      antenna, for this first run. You are testing the software's
      behavior, not making a contact.
- [ ] **Second receiver.** Have a separate receiver (another radio, a
      websdr, anything not sharing this station's antenna path) so you can
      independently confirm what actually goes out, separate from what
      `talk` believes it sent.
- [ ] **Rehearsal-soak first.** Run `talk` unarmed against the real radio
      (RX-only, always safe) until the VAD threshold and wake matcher are
      behaving well on real band noise. Arming a session that mis-hears
      constantly is a bad first experience and a waste of the exercise.
- [ ] **A quiet channel/frequency**, ideally one with no other traffic, so
      early runs don't step on anyone.
- [ ] Both Thetis servers enabled and reachable: CAT (`13013`) and TCI
      (`50001`).
- [ ] `talk/config.toml` reviewed — especially `scripts.id_text`,
      `scripts.signoff`, and the budgets under `[budgets]`.

## Step 1: the keypress-kill drill, before anything else

Arm the session (command below), then — before ever saying anything that
would trigger a reply — press any key. Confirm:

- The terminal prints the kill message and the session exits.
- The transmitter, if it happened to be mid-transmission (it won't be yet,
  since nothing triggered a reply — but if you want to test this path
  specifically, wait for a real transmission in Step 2 and kill mid-reply
  instead), shows a confirmed unkey, not a hang.

Do not proceed to Step 2 until this drill has worked cleanly at least once.

## Step 2: arm it

Run `talk/tests/live_armed.sh` — it checks the two required opt-ins
(`TALK_HOST` and `THETIS_LIVE_ALLOW_TX`, mirroring thetisctl's own live-TX
test gate) and, if both are present and you're at a real terminal, prints
the exact command. It does not run anything TX-capable itself:

```bash
export TALK_HOST=<radio-ip>
export THETIS_LIVE_ALLOW_TX=I-UNDERSTAND-THIS-KEYS-THE-RADIO
talk/tests/live_armed.sh
```

Then run the command it prints, yourself, at your own terminal:

```bash
talk/.venv/bin/python -m talk --config talk/config.toml \
    --armed --confirm-tx I-UNDERSTAND-THIS-KEYS-THE-RADIO
```

## Step 3: trigger a reply

Speak your station's phonetic callsign (or a configured wake name) toward
the radio — over a handheld, a second station, whatever you have — and
confirm:

- The transcript and trigger decision appear in the terminal and the
  session log.
- The reply is composed, synthesized, and transmitted.
- The station ID is present in the first transmission.
- On the second receiver, what you actually hear matches what the log
  says was sent.

## Step 4: end it deliberately

Either let the QSO idle-close (2 minutes of silence by default) or press a
key to kill the session outright. Confirm the process exits cleanly and the
radio is left unkeyed (check `thetisctl cat status` from another terminal
if in doubt).

## If anything looks wrong

Kill it (any key) first, ask questions after. `talk` disarms on its own for
frequency/mode drift, a stalled RX link, or an unconfirmed unkey — but the
keypress kill switch is the operator's own backstop and doesn't depend on
`talk` noticing anything itself.
