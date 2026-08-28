# Command reference

Every command `thetisctl` currently implements, by protocol tier. See
[`README.md`](README.md) for install/build steps, global flags, and the TX
safety protocol; see [`PROTOCOLS.md`](PROTOCOLS.md) for the full universe of
commands each protocol *defines*, including the ones not wired here yet.

Two commands take no protocol: `thetisctl help` prints usage, and
`thetisctl version` (also `-v` / `--version`) prints `thetisctl 0.1.0` —
this CLI's own release version, distinct from the `cat version` command
below, which reports the version of the Thetis software it's talking to.

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
| `quickplay on` | **TX-capable** — see [Transmitting](README.md#transmitting-tx-capable-commands); injects `Music\Thetis\quickrecord\SDRQuickAudio.wav` as RX I/Q ahead of the antenna, which is how FreeDV/RADE decode tests can be triggered remotely, but the underlying Thetis function can also key MOX for real — see [`NOTES.md`](NOTES.md) |
| `quickrec get` | Read whether Quick Rec is active |
| `quickrec on\|off` | Quick Rec: record RX audio to that same fixed file |
| `freedv get\|on\|off` | Enable/disable/read FreeDV RX decode (`fdv.c`), RX1 only |
| `freedv status` | Read FreeDV sync + SNR — read-only, e.g. `SYNC  SNR 15.3 dB` or `no sync` |
| `radae get\|on\|off` | Enable/disable/read RADE V1 RX decode, RX1 only |
| `radae status` | Read RADE sync + SNR — read-only, same format as `freedv status` |
| `radae-sanity` | **TX-capable** — see [Transmitting](README.md#transmitting-tx-capable-commands); scripts a full RADE off-air sanity check (tune, enable decode, inject a bench test signal via Quick Play, poll sync/SNR, clean up) and prints a summary — see [`NOTES.md`](NOTES.md) |
| `tciserver get` | Read whether Thetis's TCI server is listening |
| `tciserver on\|off` | Enable/disable the TCI server — works even when TCI itself is unreachable (CAT doesn't depend on it being up), so it can bootstrap TCI back on after a restart left the Setup checkbox unchecked |
| `status` | Rig ID + frequency/mode/RIT/XIT/split/TX in one call |
| `version` | Software version string, incl. the git short SHA the running build was made from (`ZZZV`) — verifies which commit a remote instance is actually running |
| `query <CODE>` | Raw read passthrough for any CAT command without a dedicated wrapper (e.g. `query ZZZV`) |
| `set <CODE> [params...]` | Raw write passthrough — refuses `TX`/`RX`/`ZZTX`/`ZZTU` (use `ptt`/`tune`, which apply the TX safety gate) |
| `zz list [prefix]` | List every registered `ZZxx` extended CAT command (~200), each classified bool/unsigned/signed/action, with valid range where known |
| `zz get <CODE>` / `zz set <CODE> <value>` | Validated get/set for any registered `ZZxx` command — the general-purpose way to reach the ~200 extended commands without a dedicated named wrapper; full list and methodology in [`PROTOCOLS.md`](PROTOCOLS.md) |
| `ptt on\|off` | **TX-capable** — see [Transmitting](README.md#transmitting-tx-capable-commands) |
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
