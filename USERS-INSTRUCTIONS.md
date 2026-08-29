# Setting up the AI voice operator (`talk`), end to end

`talk` is an AI that hears voice over your radio and answers by voice over
your radio, on analog SSB/FM, while **you** sit at the control point and
supervise. This guide takes you from nothing to an on-air armed session.

Two things to understand before you start:

- **It is a responder, not a caller.** `talk` speaks only when a station
  addresses you (your callsign in phonetics, or a configured wake word). It
  never calls CQ and never joins someone else's QSO.
- **On-air is a deliberate, human-only act.** By default `talk` runs in
  *rehearsal mode* and never keys the radio. Transmitting for real needs an
  explicit arming ritual that you perform yourself at a real terminal
  (Section 7). No agent, script, or automation arms it for you.

The design rationale and glossary live in
[`talk/README.md`](talk/README.md); this file is the how-to.

---

## 1. Prerequisites

**On the Thetis PC** (once): open **Setup** and enable both servers —

- **TCP/IP CAT Server** — default `127.0.0.1:13013`
- **TCI Server** — default `127.0.0.1:50001`

If `talk` runs on a different machine (the normal case), rebind both from
`127.0.0.1` to the Thetis box's LAN IP. Neither protocol has
authentication — keep them on a trusted LAN, never the internet. Details:
[`README.md`](README.md#enabling-the-servers-in-thetis).

**`thetisctl` on `PATH`** — `talk` drives it as a child process. Build and
link it per [`README.md`](README.md#build) (`which thetisctl` should
resolve).

**The machine running `talk`** needs Linux with ALSA (`aplay`/`arecord`,
package `alsa-utils`) for rehearsal audio, and Python — 3.12 or 3.13 are
safest (`faster-whisper` and `piper-tts` ship prebuilt wheels for those
before newer releases; `setup.sh` picks the oldest it finds).

**A licence** covering the band, mode, and power you intend to transmit
with. `talk` does not check this; you do.

---

## 2. Install

```bash
cd talk
./setup.sh
```

`setup.sh` is safe to re-run. It creates `talk/.venv`, installs
`requirements.txt` (numpy, faster-whisper, piper-tts, anthropic), and
downloads the Whisper *small* model and the Piper *en_US-lessac-medium*
voice into `talk/models/` (both gitignored). If `pip install` fails on
wheels, install Python 3.12 and re-run.

---

## 3. Configure

```bash
cp talk/config.minimal.toml talk/config.toml
$EDITOR talk/config.toml
```

Set the five required keys plus your name/QTH:

| Key | What |
|---|---|
| `[radio] host` | the Thetis PC's address |
| `[station] callsign` | transmissions go out under this |
| `[station] phonetic_words` | your call in spoken phonetics — the words people actually say on air |
| `[station] operator_name`, `qth` | used in spoken name/QTH replies |
| `[scripts] id_text` | station ID, sent automatically on the regulatory schedule |
| `[scripts] signoff` | sent when a QSO or the armed session times out |

Everything else (ports, VAD, budgets, logging) has a default — see the
fully annotated [`talk/config.toml.example`](talk/config.toml.example) and
add the matching `[section]` to override.

Validate:

```bash
talk/.venv/bin/python -m talk --config talk/config.toml --check
```

### The AI reply brain (optional but the point)

Without it, `talk` answers only from canned rules (signal report, "are you
there", name/QTH, "say again") and says "please say again" to anything
else. To enable conversational replies:

1. Get an Anthropic API key (`sk-ant-api03-…`) from
   <https://console.anthropic.com> → **API Keys**.
2. Put it in a file **outside the repo**, readable only by you:
   ```bash
   printf '%s' 'sk-ant-api03-...' > ~/.anthropic_key
   chmod 600 ~/.anthropic_key
   ```
3. Pass it in the environment when you launch `talk` (examples below).

The model is `claude-sonnet-5` (`[claude] model` in the config) — good
latency for a 20-second voice turn-around. Replies still route rules-first;
Claude only sees what the rules don't match, under a fixed station persona,
and the loop degrades back to rules if the API is slow or unreachable.

---

## 4. Test it off-air (rehearsal mode)

Rehearsal mode runs the whole pipeline — listen, transcribe, recognise,
compose, synthesize, **play the reply on local speakers** — and never keys
the radio. Do this until it behaves before you even think about arming.

### 4a. Against the real radio (RX only — always safe)

```bash
ANTHROPIC_API_KEY="$(cat ~/.anthropic_key)" \
  talk/.venv/bin/python -m talk --config talk/config.toml
```

You tune the radio. `talk` streams its RX audio, and when a station says
your phonetic callsign it replies on your speakers. To trigger it yourself
without a second operator, key a handheld into a dummy load nearby and
speak your call.

### 4b. Straight into the computer's microphone (no radio)

Point the RX stream at a live mic capture instead:

```bash
ANTHROPIC_API_KEY="$(cat ~/.anthropic_key)" \
TALK_STREAM_CMD="arecord -q -f FLOAT_LE -c 1 -r 24000 -t raw -D default" \
  talk/.venv/bin/python -m talk --config talk/config.toml
```

The `-r` rate **must match** `[radio] sample_rate` in the config (24000 by
default). If `aplay` isn't your output path, set
`TALK_PLAYER_CMD="paplay …"` (the reply WAV path is appended).

### What to say

The matcher needs **at least three of your phonetic words, in order**,
within a short window — or a configured `wake_names` entry. Notes from real
use:

- Whisper often writes "five" as the digit **5**, which doesn't fuzzy-match
  "five". Say the **full** call ("whiskey five tango sierra uniform") so
  three *words* still land, or drop to **"whiskey tango uniform"**.
- Saying the letters "W-5-T-S-U" does **not** work — the matcher keys on
  the phonetic words, not the callsign string.
- Then ask your question. Canned patterns answer instantly:
  "how do you copy", "are you there", "your QTH" / "where are you", "say
  again". Anything else goes to Claude (needs the key).

Every session writes `talk/logs/<timestamp>/` — a `session.jsonl` with each
utterance's transcript, trigger decision and score, plus RX/reply WAVs.
Read it to see what `talk` actually heard.

> An off-air *call-CQ-then-AI-reply* demo exists as a standalone script
> (not part of `talk`, local audio only). Ask if you want it packaged.

---

## 5. Common setup snags

From actual bring-up on a fresh box:

- **Mic muted at the OS mixer.** `amixer -c0 sget Capture` shows `[off]` →
  `amixer -c0 sset Capture cap`, then raise the level
  (`amixer -c0 sset Capture 85%`) and mic boost
  (`amixer -c0 sset 'Mic Boost' 2`).
- **No reply audio.** Output side is muted too:
  `amixer -c0 sset Master unmute; amixer -c0 sset Headphone unmute`.
- **Mic goes intermittent** between runs — PipeWire suspends or re-mutes a
  shared source. Capture the hardware directly:
  `-D plughw:CARD=<card>,DEV=0` instead of `-D default`.
- **`WARNING radio delivers 48000 Hz, not the requested 24000`** — benign.
  `talk` detects and uses the real rate.
- **Nothing transcribes.** Confirm the VAD is endpointing at all (check
  `session.jsonl`); speak louder/closer, or lower `[vad] threshold_ratio`.

---

## 6. Tune the VAD before arming

Run rehearsal against the real radio on a normal band and watch
`talk/logs/<ts>/session.jsonl`:

- `threshold_ratio` too low → band noise triggers phantom utterances.
- Too high → the first syllable of real speech is clipped before onset
  confirms.
- `hangover_ms` too short → one sentence splits into several utterances.

Adjust `[vad]` in `config.toml`, re-run, compare. A wake matcher that
mis-hears constantly makes for a bad first armed session.

---

## 7. Going ON-AIR (armed)

**Read [`talk/tests/live_armed.md`](talk/tests/live_armed.md) in full
first.** It is the procedure; this is the summary.

### The consent model

One arming action authorises a **bounded session** of repeated automated
transmissions. What keeps that safe:

- **You run the arming command yourself, at a real terminal.** `talk`
  refuses to arm without a TTY, because the kill switch needs one.
- **Any keypress or Ctrl-C** unkeys immediately (with a *confirmed* unkey,
  not fire-and-forget) and disarms.
- **Hard budgets in code:** ≤ 60 s per transmission, ≤ 10 min per QSO then
  a scripted signoff, an overall session expiry, and automatic disarm if
  the frequency/mode changes under it, the RX link dies, or a transmission
  can't confirm it unkeyed.
- **Station ID** is sent automatically (first transmission, every 10
  minutes, at signoff); the *text* is your `scripts.id_text`.
- It is still a **responder** — armed, it answers stations that call your
  callsign. It will not call CQ.

### Before the first armed run

- [ ] Transmitter into a **dummy load**, not an antenna.
- [ ] A **separate receiver** (second radio, WebSDR) to confirm
      independently what actually went out.
- [ ] Rehearsal-soaked (Sections 4–6) until wake/VAD behave on real noise.
- [ ] A **quiet frequency** within your licence privileges.
- [ ] Both Thetis servers reachable; `config.toml` scripts and `[budgets]`
      reviewed.

### The steps

1. Do the **keypress-kill drill** from `live_armed.md` Step 1 — arm, then
   press a key before anything triggers a reply, and confirm the session
   exits and the radio is unkeyed. Repeat until clean.
2. Set the two independent opt-ins and run the gate:
   ```bash
   export TALK_HOST=<radio-ip>
   export THETIS_LIVE_ALLOW_TX=I-UNDERSTAND-THIS-KEYS-THE-RADIO
   talk/tests/live_armed.sh
   ```
   It checks the opt-ins and the TTY and **prints** the exact
   `--armed --confirm-tx` command — it never runs it. You run it.
3. Trigger a reply: say your phonetic callsign toward the radio. Confirm
   the transcript, the composed reply, the ID in the first transmission,
   and — on the second receiver — that what went out matches the log.
4. End deliberately: let the QSO idle-close (2 min silence, default) or
   press a key. Verify the radio is unkeyed:
   ```bash
   thetisctl cat --host <radio-ip> status
   ```

If anything looks wrong: kill it first (any key), diagnose after.

---

## 8. Where things live

| Path | What |
|---|---|
| `talk/config.toml` | your station config (gitignored) |
| `talk/config.minimal.toml` / `.example` | templates |
| `talk/.venv/`, `talk/models/` | environment + speech models (gitignored) |
| `talk/logs/<timestamp>/` | per-session transcript (`session.jsonl`) + WAVs |
| `talk/README.md` | design record, glossary, decisions |
| `talk/tests/live_armed.md` | the human-only on-air procedure |
| `~/.anthropic_key` | your API key (you create this, outside the repo) |
