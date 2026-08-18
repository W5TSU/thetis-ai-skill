# TCI & CAT command reference

Full command inventories for the two protocols `thetisctl` speaks, generated
directly from source rather than from memory, so this file can be diffed
against the upstream sources and re-generated when they change. See
`README.md` for the curated, day-to-day command tables and usage examples;
this file is the exhaustive cross-reference behind them.

## Sources

| Protocol | Source | Version / commit |
|---|---|---|
| TCI | Expert Electronics' "Universal transceiver control interface – TCI" spec (`doc/TCI_interface_1.6.pdf`, [maksimus1210/TCI](https://github.com/maksimus1210/TCI)), cross-checked against Thetis's own `Project Files/Source/Console/TCIServer.cs` | spec v1.6, 2021; TCIServer.cs at commit `4969b62f` (2026-08-16) |
| CAT | Thetis's own `Project Files/Source/Console/CAT/{CATStructs.xml,CATCommands.cs,CATParser.cs}` | commit `4969b62f` (2026-08-16), [W5TSU/OpenHPSDR-Thetis-Hermes-Lite2](https://github.com/W5TSU/OpenHPSDR-Thetis-Hermes-Lite2) |

Thetis's TCI server (`TCIServer.cs`) implements this same TCI protocol, plus
a handful of Thetis-only extensions not in the upstream spec (noted below).
Its CAT server implements a Kenwood TS-2000-style command set (the 107
"legacy" 2-letter codes) extended with ~320 Thetis/FlexRadio-style `ZZxx`
4-letter codes — CAT has no single upstream spec to diff against the way TCI
does, so its table below is generated straight from Thetis's own
`CATStructs.xml`, cross-checked against `CATParser.cs`'s actual dispatch
switch (see "Dispatched" column) rather than trusted at face value — see the
callout under [CAT gotchas](#cat-gotchas-found-while-generating-this-table).

**"thetisctl" column**: the exact `thetisctl` subcommand that sends this
wire command, where one exists. Blank means no dedicated CLI subcommand
exists for it today — for TCI it's still reachable via
`thetisctl tci query <cmd> [args...]` (raw passthrough, works for get *and*
set forms); for CAT, `thetisctl cat query <CODE>` (read) and
`thetisctl cat set <CODE> [params...]` (write) cover the same ground —
except `TX`/`RX`/`ZZTX`/`ZZTU`, which both refuse to send since those key
the transmitter (use `ptt`/`tune` instead).

## Summary

| | Documented | Implemented in thetisctl |
|---|---|---|
| TCI | 80 commands (v1.0 base + v1.4/v1.5/v1.6 additions) | 52 (~65%), plus 5 Thetis-only extensions not in the spec at all — of the remaining 28, 27 don't exist in Thetis's own `TCIServer.cs` at all (see below) and 1 (`TX_FREQUENCY`) is a push-only broadcast already fully reachable via `tci query` |
| CAT | 429 commands (107 legacy Kenwood-style + 322 `ZZxx` extended) | 220 (~51%) — 22 named commands + 197 `ZZxx` fields reachable through the generic, validated `cat zz get/set` + 1 (`ZZTU`) via the new safety-gated `cat tune`. Scope was deliberately `ZZxx`-only (see below); the 107 legacy Kenwood commands remain untouched beyond the original 7. |

## TCI commands

`RO`/`WO`/`RW` = read-only / write-only / read-write, per the spec. "Added"
is the TCI spec version each command first appeared in. The five `cw_*`/
`callsign_send` rows are documented in the spec's prose (the CW-macro
section) rather than in its command table, but are real wire commands like
everything else here.

**Total documented commands: 80; implemented with a typed thetisctl command: 52.**

| Command | Type | Added | Description | thetisctl |
|---|---|---|---|---|
| `VFO_LIMITS` | RO | v1.0 | Receiver's frequency tuning limits |  |
| `IF_LIMITS` | RO | v1.0 | IF filter frequency limits |  |
| `TRX_COUNT` | RO | v1.0 | Number of receivers/transceivers |  |
| `CHANNEL_COUNT` | RO | v1.0 | Number of additional receiver channels (slices) |  |
| `DEVICE` | RO | v1.0 | Name of the device |  |
| `RECEIVE_ONLY` | RO | v1.0 | Determine device as a receiver or transceiver |  |
| `MODULATIONS_LIST` | RO | v1.0 | List of supported mode types |  |
| `TX_ENABLE` | RO | v1.0 | Permission to use transmitter |  |
| `READY` | RO | v1.0 | Sent after initialization commands while connecting |  |
| `TX_FOOTSWITCH` | RO | v1.0 | PTT footswitch signal |  |
| `START` | WO | v1.0 | Start ExpertSDR2/Thetis engine | `tci power on` |
| `STOP` | WO | v1.0 | Stop ExpertSDR2/Thetis engine | `tci power off` |
| `DDS` | RW | v1.0 | Tuning of the RX's center frequency (panorama center) | `tci dds` |
| `IF` | RW | v1.0 | IF filter tuning in panorama bandwidth | `tci if-offset` |
| `RIT_ENABLE` | RW | v1.0 | Enable RIT | `tci rit` |
| `MODULATION` | RW | v1.0 | Set mode type | `tci modulation` |
| `RX_ENABLE` | RW | v1.0 | Enable software receivers | `tci rx-enable` |
| `XIT_ENABLE` | RW | v1.0 | Enable XIT | `tci xit` |
| `SPLIT_ENABLE` | RW | v1.0 | Enable SPLIT mode | `tci split` |
| `RIT_OFFSET` | RW | v1.0 | Tune RIT offset | `tci rit-offset` |
| `XIT_OFFSET` | RW | v1.0 | Tune XIT offset | `tci xit-offset` |
| `RX_CHANNEL_ENABLE` | RW | v1.0 | Enable additional receive channels | `tci rx-channel` |
| `RX_FILTER_BAND` | RW | v1.0 | Adjust IF filter width | `tci filter` |
| `RX_SMETER` | RW | v1.0 | Signal level (S-Meter) in filter bandwidth |  |
| `CW_MACROS_SPEED` | RW | v1.0 | Set CW speed for macros | `tci cw send --speed` |
| `CW_MACROS_SPEED_UP` | WO | v1.0 | Increase CW speed for macros | `tci cw speed-up` |
| `CW_MACROS_SPEED_DOWN` | WO | v1.0 | Decrease CW speed for macros | `tci cw speed-down` |
| `CW_MACROS_DELAY` | RW | v1.0 | Delay between "turn to TX" and "start of macro transmission" | `tci cw delay` |
| `TUNE` | RW | v1.0 | Switch RX/TUNE modes | `tci tune (TX)` |
| `IQ_START` | RW | v1.0 | Start IQ signal output | `tci iq capture/stream` |
| `IQ_STOP` | RW | v1.0 | Stop IQ signal output | `tci iq capture/stream` |
| `IQ_SAMPLERATE` | RW | v1.0 | Set IQ signal sample rate | `tci iq-samplerate` |
| `AUDIO_START` | RW | v1.0 | Start audio stream | `tci rx-audio (internal)` |
| `AUDIO_STOP` | RW | v1.0 | Stop audio stream | `tci rx-audio (internal)` |
| `AUDIO_SAMPLERATE` | RW | v1.0 | Set audio stream sample rate | `tci audio-samplerate` |
| `SPOT` | WO | v1.0 | Send spot to display | `tci spot send` |
| `SPOT_DELETE` | WO | v1.0 | Delete a spot | `tci spot delete` |
| `SPOT_CLEAR` | WO | v1.0 | Clear all spots | `tci spot clear` |
| `PROTOCOL` | RO | v1.0 | TCI protocol version, sent on connect |  |
| `TX_POWER` | RO | v1.0 | Output power level, W |  |
| `TX_SWR` | RO | v1.0 | Transmitter SWR value |  |
| `VOLUME` | RW | v1.0 | Main software volume | `tci volume` |
| `SQL_ENABLE` | RW | v1.0 | On/off squelch | `tci sql` |
| `SQL_LEVEL` | RW | v1.0 | Squelch threshold | `tci sql-level` |
| `VFO` | RW | v1.0 | Set receiver's tuning frequency | `tci vfo` |
| `APP_FOCUS` | RO | v1.0 | Status of the main app window (in focus or not) |  |
| `SET_IN_FOCUS` | WO | v1.0 | Set the main app window in focus | `tci focus` |
| `MUTE` | RW | v1.0 | Mute — disable/enable overall volume | `tci mute` |
| `RX_MUTE` | RW | v1.0 | Mute a certain receiver | `tci rx-mute` |
| `CTCSS_ENABLE` | RW | v1.4 | Enable/disable CTCSS tones |  |
| `CTCSS_MODE` | RW | v1.4 | Switch CTCSS tone modes |  |
| `CTCSS_RX_TONE` | RW | v1.4 | Set CTCSS tone for a receiver |  |
| `CTCSS_TX_TONE` | RW | v1.4 | Set CTCSS tone for a transmitter |  |
| `CTCSS_LEVEL` | RW | v1.4 | Control CTCSS tone level in NFM mode |  |
| `ECODER_SWITCH_RX` | RW | v1.4 | Switch control over active RX with E-Coder panel |  |
| `ECODER_SWITCH_CHANNEL` | RW | v1.4 | Switch control over active channel with E-Coder panel |  |
| `RX_VOLUME` | RW | v1.4 | Volume control for each channel in software receivers | `tci rx-volume` |
| `RX_BALANCE` | RW | v1.4 | Volume balance control for each channel in software receivers | `tci rx-balance` |
| `TRX` | RW | v1.5 | Switch between RX/TX modes | `tci ptt (TX)` |
| `DRIVE` | RW | v1.5 | Control transmitter power output | `tci drive` |
| `TUNE_DRIVE` | RW | v1.5 | Control transmitter power output in TUNE mode | `tci tune-drive` |
| `RX_SENSORS_ENABLE` | WO | v1.5 | Enable/disable sharing of S-meter readings | `tci rx-sensors` |
| `TX_SENSORS_ENABLE` | WO | v1.5 | Enable/disable sharing of transmitter readings | `tci tx-sensors` |
| `RX_SENSORS` | RO | v1.5 | Shared RX signal level (in filter bandwidth) |  |
| `TX_SENSORS` | RO | v1.5 | Shared TX signal parameters (mic level, RMS/PEAK power, SWR) |  |
| `RX_NB_ENABLE` | RW | v1.6 | Enable/disable Noise Blanker (NB) | `tci nb` |
| `RX_NB_PARAM` | RW | v1.6 | Adjust Noise Blanker (NB) parameters |  |
| `RX_BIN_ENABLE` | RW | v1.6 | Enable/disable pseudo stereo (BIN) | `tci bin` |
| `RX_NR_ENABLE` | RW | v1.6 | Enable/disable noise reduction (NR) | `tci nr` |
| `RX_ANC_ENABLE` | RW | v1.6 | Enable/disable Automatic Noise Cancellation (ANC) |  |
| `RX_ANF_ENABLE` | RW | v1.6 | Enable/disable Automatic Notch Filter (ANF) | `tci anf` |
| `RX_APF_ENABLE` | RW | v1.6 | Enable/disable Analogue Peak Filter (APF) | `tci apf` |
| `RX_DSE_ENABLE` | RW | v1.6 | Enable/disable Digital Surround Effect (DSE) for CW |  |
| `RX_NF_ENABLE` | RW | v1.6 | Enable/disable band Notch Filters (NF) | `tci nf` |
| `TX_FREQUENCY` | RO | v1.6 | Transmitter frequency, Hz |  |
| `cw_macros` | WO | v1.0 | Send free-text CW macro (periodic-number + text form) | `tci cw send (TX)` |
| `cw_terminal` | WO | v1.5 | Enable/disable CW terminal mode (stay in TX after macro finishes) | `tci cw terminal` |
| `cw_msg` | WO | v1.0 | Send CW message with callsign-repeat / mid-transmission-edit support | `tci cw send-msg (TX) / cw edit-callsign` |
| `callsign_send` | WO | v1.0 | Edit the callsign of an in-progress cw_msg transmission | `tci cw edit-callsign` (Thetis has no `callsign_send` wire command — this functionality lives in `cw_msg`'s 1-arg form instead, see below) |
| `cw_macros_stop` | WO | v1.0 | Abort in-progress CW macro transmission and unkey | `auto-sent on unkey` |

### Spec commands Thetis does not implement

28 of the v1.6 spec's 80 commands have **zero occurrences** of their wire
token anywhere in this Thetis checkout's `TCIServer.cs` (grepped directly,
2026-08-17) — not stubbed, not disabled, simply never written. Sending any
of these to a real Thetis instance is a silent no-op (unrecognized command,
ignored). `thetisctl` deliberately does not implement typed wrappers for
these — there is nothing on the wire to wrap:

`VFO_LIMITS`, `IF_LIMITS`, `TRX_COUNT`, `CHANNEL_COUNT`, `DEVICE`,
`RECEIVE_ONLY`, `MODULATIONS_LIST`, `TX_ENABLE`, `READY`, `TX_FOOTSWITCH`,
`RX_SMETER`, `PROTOCOL`, `TX_POWER`, `TX_SWR`, `APP_FOCUS`, `RX_SENSORS`,
`TX_SENSORS`, `RX_NB_PARAM`, `CTCSS_ENABLE`, `CTCSS_MODE`, `CTCSS_RX_TONE`,
`CTCSS_TX_TONE`, `CTCSS_LEVEL`, `ECODER_SWITCH_RX`, `ECODER_SWITCH_CHANNEL`,
`RX_ANC_ENABLE`, `RX_DSE_ENABLE`, `callsign_send` (see its row above — its
functionality survives inside `cw_msg`, just not as its own command).

Most of the "to be sent after connection"-only fields (`VFO_LIMITS` through
`TX_FOOTSWITCH`, `PROTOCOL`, `APP_FOCUS`) are a distinct case from the
CTCSS/E-Coder/sensor ones: even in a *compliant* TCI v1.6 implementation
these have no client-request form at all (spec's own "Read: To be sent
after connection" row) — they're server-push-only by design, so their
absence from Thetis's incoming-command dispatch switch doesn't mean Thetis
skipped implementing a request handler, only that it (like the spec
intends) never pushes them because it has nothing to say (Thetis's TCI
identifies itself via a bespoke non-spec `run_cat_ex`-adjacent path instead
of `DEVICE`/`PROTOCOL`). `TX_FREQUENCY` is the one exception that Thetis
*does* push (see the main table) despite also having no request form.

### Thetis-only TCI extensions (not in the upstream spec)

Confirmed against `TCIServer.cs` directly (see `internal/tci/control.go`'s
doc comments for exact line numbers) — these are Thetis additions to the
base TCI protocol, not part of the v1.6 spec above:

| Wire command | thetisctl | Purpose |
|---|---|---|
| `rx_step_att_ex` | `tci atten` | RX step attenuator, dB |
| `rx_preamp_att_ex` | `tci preamp` | RX preamp gain, expressed as ≤0 dB attenuation |
| `agc_mode` | `tci agc` | AGC mode (off/fixed/long/slow/medium/fast/custom) |
| `agc_gain` | `tci agc-gain` | AGC threshold/gain |
| `audio_stream_sample_type` | internal (`rx-audio`/`tx-audio`) | Negotiates int16/int24/int32/float32 encoding for audio stream frames |

**Known but not catalogued or implemented** — found while reading
`TCIServer.cs`'s dispatch switch (~line 5294 onward) for the spec commands
above, not yet individually verified/wired: `lock`, `vfo_lock` (per-VFO
lock, unrelated to CAT's VFO lock naming), `keyer` (raw manual paddle-press
keying — **would be TX-capable**), `mon_volume`, `mon_enable` (TX monitor
audio), `line_out_start`/`line_out_stop` (VAC line-out), `spot_simulate_click`,
`digl_offset`/`digu_offset` (digital-mode click-tune offsets), `cw_keyer_speed`
(distinct from `cw_macros_speed`), `vfo_sync_ex`, `vfo_swap_ex`,
`fm_deviation_ex`, `agc_auto_ex`, `rx_ctun_ex`, `tx_profile_ex`,
`tx_profiles_ex`, `calibration_ex`, `run_cat_ex`, `shutdown_ex`,
`audio_stream_channels`, `audio_stream_samples`, `tx_stream_audio_buffering`,
`rx_nb_enable_ex`/`rx_nr_enable_ex` (NB/NR-algorithm-select variants of the
two already-implemented plain forms), `rx_step_att_enabled_ex`. Treat this
list as a lead for future work, not a verified inventory — confirm each
against its handler body before wiring, same as everything else in this
file.

## CAT commands

All 429 commands defined in `CATStructs.xml` (357 marked `<active>true</active>`,
71 `false`), split into the 107 legacy 2-letter (Kenwood TS-2000-compatible)
codes and the 322 `ZZxx` Thetis/FlexRadio-style extended codes. "`<active>`" is that struct's own
flag (Thetis's own marker for whether a command is considered live vs.
reserved/stubbed); "Dispatched" is whether `CATParser.cs`'s dispatch switch
actually routes that code to a handler — generated by diffing
`CATStructs.xml`'s command list against `CATParser.cs`'s `case` labels
directly, not assumed from `<active>`.

### How the 197 `ZZxx` fields were implemented (2026-08-17)

Scope was deliberately narrowed to the 322 `ZZxx` extended commands (not the
107 legacy Kenwood codes) — that's where Thetis's own modern control surface
lives; the legacy set is mostly satellite/DRU-3A/packet-cluster/voice-memory
TS-2000 features with no HL2/Thetis equivalent. Given the volume (~300
commands vs. TCI's ~30), this was **not** done by hand-verifying each one the
way the TCI work was — it was done by cross-referencing every command's
*actual handler body* in `CATCommands.cs` against its `CATStructs.xml`
struct entry, programmatically classifying each into one of four wire
shapes, then generating both the Go registry (`internal/cat/zzfields.go`)
and this table from that classification — auditable and re-runnable, not
guessed from parameter counts alone:

- **`bool`** (77 fields) — `"0"`/`"1"`, one wire byte.
- **`unsigned`** (45 fields) — a fixed-width zero-padded decimal, e.g. `ZZAG`
  (extended AF Gain) is `042` for 42%.
- **`signed`** (25 fields) — one sign character (`+`/`-`) followed by a
  fixed-width zero-padded magnitude, e.g. `ZZAA` (APF gain) is `+005` for 5%
  — confirmed against `AddLeadingZeros`' exact padding convention
  (`CATCommands.cs:8884-8902`), not assumed.
- **`action`** (50 fields) — no parameters, e.g. `ZZBA` ("RX2 down one
  band"): sending the bare code performs the action.

Each field's valid range (where the handler clamps one via
`Math.Max`/`Math.Min`) was extracted from the same handler body and is
enforced client-side before sending. All four shapes are reachable through
one CLI surface — `thetisctl cat zz list|get|set` — rather than ~200
individual named subcommands; see `internal/cat/zzext.go`.

**Two commands were confirmed to genuinely key the transmitter** despite
fitting the plain `bool` shape, and are deliberately excluded from the
registry so no generic passthrough can reach them: `ZZTX` ("MOX button", sets
`console.CATPTT`) and `ZZTU` ("TUNE" — despite its description not saying
so, it sets `console.TUN`, a bare carrier, exactly like TCI's `TUNE`).
`ZZTU` got a proper safety-gated wrapper (`thetisctl cat tune`, mirroring
`tci tune`'s dry-run/confirm-tx/auto-unkey/5s-cap shape); `ZZTX` didn't need
one since the existing `cat ptt` already keys PTT via the legacy bare
`TX`/`RX` commands. Both are also blocked in the raw `cat set` passthrough.
This is exactly the kind of thing hand-classification-at-scale can miss —
worth remembering before extending this table further.

**~103 `ZZxx` commands were left out of the registry** — their handler
bodies don't fit the four generic shapes above (composite/multi-field
replies, enums, or free-text), or they're out of scope for a remote CLI:

- **Composite/enum-encoded** (worth doing next, if this table is extended):
  antenna control (`ZZOA`-`ZZOJ`: RX/TX antenna select, ext-ant, relay
  enable/delay), meters (`ZZMR`/`ZZMT`/`ZZRM`, read-only), extended
  mode/display enums (`ZZMD`, `ZZDM`, `ZZOD`), VFO A/B direct (`ZZFA`/`ZZFB`),
  FM deviation (`ZZFD`), step-nudge commands (`ZZAD`/`ZZAE`/`ZZAF`/`ZZBE`/
  `ZZBF`/`ZZBM`/`ZZBP`/`ZZRD`/`ZZRU`/`ZZXD`/`ZZXU`, all "move N steps" with a
  slightly different shape than the plain numeric fields above), CWX
  send/macro (`ZZKY`/`ZZKM`, free text).
- **Out of scope / low value for a remote CLI**: mixer/VAC audio routing
  (`ZZWA`-`ZZWS`, ~20 commands, sound-card-specific), F1500-specific mixer
  controls (`ZZWT`-`ZZWW`), FlexWire (`ZZFV`-`ZZFY`), front-panel hardware
  (`ZZZD`/`ZZZE`/`ZZZP`/`ZZZS`/`ZZZU`/`ZZZW`), Aries ATU (`ZZOV`-`ZZOZ`),
  ganymede amplifier (`ZZZA`), internal guid add/remove (`ZZGA`/`ZZGR`,
  whose own description says "used by TCPIPcat internally, ignore").
- **`ZZDY`, `ZZPK`, `ZZPL`** — see [CAT gotchas](#cat-gotchas-found-while-generating-this-table);
  their handler bodies are commented out entirely, so there's nothing to wire.

None of the ~103 above are reachable through a dedicated `thetisctl` command
today; the raw `cat set <CODE> <params>` / `cat query <CODE>` passthrough
(added alongside this work — CAT previously had `query` but no generic
write) still reaches all of them for anyone willing to hand-encode the wire
format from `CATCommands.cs` directly, same as before.

### Legacy (Kenwood-style, 2-letter) commands — 107 total

| Code | Description | `<active>` | Dispatched | thetisctl |
|---|---|---|---|---|
| `AC` | antenna tuner status | no | ✅ |  |
| `AG` | af gain | yes | ✅ |  |
| `AI` | auto information | yes | ✅ |  |
| `AL` | auto notch level | no | ✅ |  |
| `AM` | auto mode on_off | no | ✅ |  |
| `AN` | select antenna connector | no | ✅ |  |
| `AR` | asc function on_off | no | ✅ |  |
| `AS` | auto mode function | no | ✅ |  |
| `BC` | beat canceller on_off | no | ✅ |  |
| `BD` | move down band | yes | ✅ |  |
| `BP` | beat canceller frequency | no | ✅ |  |
| `BU` | move up band | yes | ✅ |  |
| `BY` | read busy signal status | no | ✅ |  |
| `CA` | cw autotune on_off | no | ✅ |  |
| `CG` | carrier gain | no | ✅ |  |
| `CH` | current freq to call channel | no | ✅ |  |
| `CI` | vfo_mem current freq to call channel | no | ✅ |  |
| `CM` | packet cluster tune on_off | no | ✅ |  |
| `CN` | ctcss function | yes | ✅ |  |
| `CT` | ctcss function status | yes | ✅ |  |
| `DC` | tx band status | no | ✅ |  |
| `DN` | mic down key | yes | ✅ |  |
| `DQ` | dcs function status | no | ✅ |  |
| `EX` | extension menu | no | ✅ |  |
| `FA` | vfo a frequency | yes | ✅ | `cat freq get/set A` |
| `FB` | vfo b frequency | yes | ✅ | `cat freq get/set B` |
| `FC` | subreceiver vfo freq | no | ✅ |  |
| `FD` | filter display dot pattern | no | ✅ |  |
| `FR` | selects rx vfo mem or call | yes | ✅ |  |
| `FS` | fine function status | no | ✅ |  |
| `FT` | selects tx vfo mem or call | yes | ✅ |  |
| `FW` | dsp filter width | no | ✅ |  |
| `GT` | agc constant status | yes | ✅ |  |
| `ID` | transceiver id number | yes | ✅ | `cat status/version` |
| `IF` | transceiver status | yes | ✅ | `cat status` |
| `IS` | if shift function | no | ✅ |  |
| `KS` | keyer speed | yes | ✅ |  |
| `KY` | convert to morse code | yes | ✅ |  |
| `LK` | key lock function | no | ✅ |  |
| `LM` | dru_3a recording status | no | ✅ |  |
| `LT` | alt function status | no | ✅ |  |
| `MC` | recall or read memory channel | no | ✅ |  |
| `MD` | operating mode | yes | ✅ | `cat mode get/set` |
| `MF` | menu a or b | no | ✅ |  |
| `MG` | microphone gain | yes | ✅ |  |
| `ML` | monitor function level | no | ✅ |  |
| `MO` | monitor on_off sky commander | yes | ✅ |  |
| `MR` | read memory channel | no | ✅ |  |
| `MU` | set memory group | no | ✅ |  |
| `MW` | store memory channel | no | ✅ |  |
| `NB` | noise blanker function on_off | yes | ✅ |  |
| `NL` | noise blanker level | no | ✅ |  |
| `NR` | noise reduction status | no | ✅ |  |
| `NT` | auto notch function on_off | yes | ✅ |  |
| `OF` | offset frequency | yes | ✅ |  |
| `OI` | read memory channel | no | ✅ |  |
| `OS` | offset function status | yes | ✅ |  |
| `PA` | preamp function status | no | ✅ |  |
| `PB` | dru_3a playback status | no | ✅ |  |
| `PC` | output power | yes | ✅ |  |
| `PI` | store progammable memory channel | no | ✅ |  |
| `PK` | read packet cluster data | no | ✅ |  |
| `PL` | speech processor io level | no | ✅ |  |
| `PM` | recall programmable memory | no | ✅ |  |
| `PR` | speech processor function on_off | yes | ✅ |  |
| `PS` | power on_off | yes | ✅ |  |
| `QC` | dcs code | no | ✅ |  |
| `QI` | store quick memory | yes | ✅ |  |
| `QR` | read quick memory | no | ✅ |  |
| `RA` | attenuator function status | no | ✅ |  |
| `RC` | clear rit frequency | yes | ✅ |  |
| `RD` | move rit offset down | yes | ✅ |  |
| `RG` | rf gain status | no | ✅ |  |
| `RL` | noise reduction level | no | ✅ |  |
| `RM` | meter function | no | ✅ |  |
| `RT` | rit function on_off | yes | ✅ |  |
| `RU` | move rit offset up | yes | ✅ |  |
| `RX` | rx function status | yes | ✅ | `cat ptt off` |
| `SA` | satellite mode status | no | ✅ |  |
| `SB` | sub or tf_w status | no | ✅ |  |
| `SC` | scan function status | no | ✅ |  |
| `SD` | cw break_in delay | no | ✅ |  |
| `SH` | dsp filter high setting | yes | ✅ |  |
| `SI` | input satellite memory name | no | ✅ |  |
| `SL` | dsp filter low setting | yes | ✅ |  |
| `SM` | s_meter status | yes | ✅ |  |
| `SQ` | squelch level | yes | ✅ |  |
| `SR` | reset transceiver | no | ✅ |  |
| `SS` | program scan pause frequency | no | ✅ |  |
| `ST` | multi_ch control frequency step | no | ✅ |  |
| `SU` | program scan pause frequency | no | ✅ |  |
| `SV` | execute memory transfer | no | ✅ |  |
| `TC` | internal tnc mode | no | ✅ |  |
| `TD` | send dtmf memory channel data | no | ✅ |  |
| `TI` | tnc led status | no | ✅ |  |
| `TN` | sub tone frequency | no | ✅ |  |
| `TO` | tone function on_off | no | ✅ |  |
| `TS` | tf_set function | no | ✅ |  |
| `TX` | set tx mode | yes | ✅ | `cat ptt on` |
| `TY` | read microprocessor firmware type | no | ✅ |  |
| `UL` | pll unlock status | no | ✅ |  |
| `UP` | microphone up key | yes | ✅ |  |
| `VD` | vox time delay | no | ✅ |  |
| `VG` | vox gain | no | ✅ |  |
| `VR` | voice1 or voice2 key | no | ✅ |  |
| `VX` | vox function status | no | ✅ |  |
| `XT` | xit function status | yes | ✅ |  |


### ZZ-extended commands — 322 total

| Code | Description | `<active>` | Dispatched | thetisctl |
|---|---|---|---|---|
| `ZZAA` | APF gain | yes | ✅ | `cat zz get/set ZZAA` |
| `ZZAB` | APF bandwidth | yes | ✅ | `cat zz get/set ZZAB` |
| `ZZAC` | Sets or reads the step size | yes | ✅ | `cat zz get/set ZZAC` |
| `ZZAD` | Moves VFOA down nn Tune Steps | yes | ✅ |  |
| `ZZAE` | Moves VFOA down nn Steps | yes | ✅ |  |
| `ZZAF` | Moves VFOA up nn Steps | yes | ✅ |  |
| `ZZAG` | extended AF Gain | yes | ✅ | `cat zz get/set ZZAG` |
| `ZZAI` | Auto Info | yes | ✅ |  |
| `ZZAP` | Audio peak filter on off | yes | ✅ | `cat zz get/set ZZAP` |
| `ZZAR` | AGCRF | yes | ✅ | `cat zz get/set ZZAR` |
| `ZZAS` | RX2 AGCT | yes | ✅ | `cat zz get/set ZZAS` |
| `ZZAT` | APF tune | yes | ✅ | `cat zz get/set ZZAT` |
| `ZZAU` | Moves VFOA up nn Tune Steps | yes | ✅ |  |
| `ZZAY` | APF type | yes | ✅ |  |
| `ZZBA` | RX2 down one band | yes | ✅ | `cat zz get/set ZZBA` |
| `ZZBB` | RX2 up one band | yes | ✅ | `cat zz get/set ZZBB` |
| `ZZBD` | Band Down | yes | ✅ | `cat zz get/set ZZBD` |
| `ZZBE` | Moves VFOB down nn Steps | yes | ✅ |  |
| `ZZBF` | Moves VFOB up nn Steps | yes | ✅ |  |
| `ZZBG` | extended Band Group | yes | ✅ | `cat zz get/set ZZBG` |
| `ZZBI` | extended BIN status | yes | ✅ | `cat zz get/set ZZBI` |
| `ZZBM` | Moves VFO B down nn Tune Steps | yes | ✅ |  |
| `ZZBP` | Moves VFO B up nn Tune Steps | yes | ✅ |  |
| `ZZBR` | BCI Reject | yes | ✅ | `cat zz get/set ZZBR` |
| `ZZBS` | extended band change | yes | ✅ | `cat band` |
| `ZZBT` | RX2Band | yes | ✅ |  |
| `ZZBU` | Band Up | yes | ✅ | `cat zz get/set ZZBU` |
| `ZZBY` | Closes the console | yes | ✅ | `cat zz get/set ZZBY` |
| `ZZCB` | Enable Break In | yes | ✅ | `cat zz get/set ZZCB` |
| `ZZCD` | CW Break In Delay | yes | ✅ | `cat zz get/set ZZCD` |
| `ZZCF` | Show CW Freq | yes | ✅ | `cat zz get/set ZZCF` |
| `ZZCI` | CW Iambic Enable | yes | ✅ | `cat zz get/set ZZCI` |
| `ZZCL` | extended CW Pitch | yes | ✅ | `cat zz get/set ZZCL` |
| `ZZCM` | CW Monitor | yes | ✅ | `cat zz get/set ZZCM` |
| `ZZCN` | CTUN Enable | yes | ✅ | `cat zz get/set ZZCN` |
| `ZZCO` | RX2 CTUN Enable | yes | ✅ | `cat zz get/set ZZCO` |
| `ZZCP` | extended Compander status | yes | ✅ | `cat zz get/set ZZCP` |
| `ZZCS` | extended CW Speed | yes | ✅ | `cat zz get/set ZZCS` |
| `ZZCT` | CPDR Threshold | yes | ✅ | `cat zz get/set ZZCT` |
| `ZZCU` | extended CPU Usage | yes | ✅ | `cat zz get/set ZZCU` |
| `ZZDA` | extended Display Average | yes | ✅ | `cat zz get/set ZZDA` |
| `ZZDB` | Diversity RX reference | yes | ✅ | `cat zz get/set ZZDB` |
| `ZZDC` | Diversity RX2 Gain | yes | ✅ | `cat zz get/set ZZDC` |
| `ZZDD` | Diversity Phase | yes | ✅ | `cat zz get/set ZZDD` |
| `ZZDE` | Diversity Form Enable | yes | ✅ | `cat zz get/set ZZDE` |
| `ZZDF` | CAT Diversity Form | yes | ✅ | `cat zz get/set ZZDF` |
| `ZZDG` | Diversity RX1 Gain | yes | ✅ | `cat zz get/set ZZDG` |
| `ZZDH` | Diversity RX Source | yes | ✅ | `cat zz get/set ZZDH` |
| `ZZDM` | extended Display mode | yes | ✅ |  |
| `ZZDN` | Waterfall Low Level | yes | ✅ | `cat zz get/set ZZDN` |
| `ZZDO` | Waterfall High Level | yes | ✅ | `cat zz get/set ZZDO` |
| `ZZDP` | Sprectrum Grid Max | yes | ✅ | `cat zz get/set ZZDP` |
| `ZZDQ` | Spectrum Grid Min | yes | ✅ | `cat zz get/set ZZDQ` |
| `ZZDR` | Spectrum Grid Step | yes | ✅ | `cat zz get/set ZZDR` |
| `ZZDS` | FreeDV RX decode sync/SNR status | yes | ✅ | `cat freedv (status)` |
| `ZZDT` | RADE RX decoder-input level/clip status | yes | ✅ | `cat zz get/set ZZDT` |
| `ZZDU` | DDUTil Status | yes | ✅ | `cat zz get/set ZZDU` |
| `ZZDV` | FreeDV RX decode enable status | yes | ✅ | `cat freedv` |
| `ZZDW` | RADE V1 RX decode enable status | yes | ✅ | `cat radae` |
| `ZZDX` | Phone DX button | yes | ✅ |  |
| `ZZDY` | DX Level | yes | ❌ **not wired** |  |
| `ZZDZ` | RADE RX decode sync/SNR status | yes | ✅ | `cat radae (status)` |
| `ZZEA` | RXEQ Values | yes | ✅ |  |
| `ZZEB` | TXEQ Values | yes | ✅ |  |
| `ZZEM` | Verbose CAT Errors | yes | ✅ | `cat zz get/set ZZEM` |
| `ZZER` | RXEQ button status | yes | ✅ | `cat zz get/set ZZER` |
| `ZZET` | TXEQ button status | yes | ✅ | `cat zz get/set ZZET` |
| `ZZFA` | VFO A | yes | ✅ |  |
| `ZZFB` | VFO B | yes | ✅ |  |
| `ZZFD` | FM Deviation | yes | ✅ |  |
| `ZZFH` | DSP Filter High | yes | ✅ | `cat zz get/set ZZFH` |
| `ZZFI` | extended current filter name | yes | ✅ | `cat zz get/set ZZFI` |
| `ZZFJ` | RX2 DSP Filter | yes | ✅ | `cat zz get/set ZZFJ` |
| `ZZFL` | DSP Filter Low | yes | ✅ | `cat zz get/set ZZFL` |
| `ZZFM` | Flex Model Number | yes | ✅ | `cat zz get/set ZZFM` |
| `ZZFR` | RX2 DSP Filter High | yes | ✅ | `cat zz get/set ZZFR` |
| `ZZFS` | RX2 DSP Filter Low | yes | ✅ | `cat zz get/set ZZFS` |
| `ZZFT` | TX Freq | yes | ✅ |  |
| `ZZFV` | FlexWire single read | yes | ✅ | `cat zz get/set ZZFV` |
| `ZZFW` | FlexWire double read | yes | ✅ | `cat zz get/set ZZFW` |
| `ZZFX` | FlexWire single | yes | ✅ |  |
| `ZZFY` | FlexWire double | yes | ✅ |  |
| `ZZGA` | Add guid - used by TCPIPcat internally, ignore | yes | ✅ |  |
| `ZZGE` | Noise Gate Enable | yes | ✅ | `cat zz get/set ZZGE` |
| `ZZGL` | Noise Gate Level | yes | ✅ | `cat zz get/set ZZGL` |
| `ZZGR` | Remove guid - used by TCPIPcat internally, ignore | yes | ✅ |  |
| `ZZGT` | extended AGC constant | yes | ✅ | `cat agc` |
| `ZZGU` | extended RX2 AGC constant | yes | ✅ |  |
| `ZZHA` | Audio Buffer Size | yes | ✅ |  |
| `ZZHR` | Phone RX Buffer | yes | ✅ |  |
| `ZZHT` | Phone TX Buffer | yes | ✅ |  |
| `ZZHU` | CW RX Buffer | yes | ✅ |  |
| `ZZHV` | CW TX Buffer | yes | ✅ |  |
| `ZZHW` | Digital RX Buffer | no | ✅ |  |
| `ZZHX` | Digital TX Buffer | yes | ✅ |  |
| `ZZID` | extended SetRigType | yes | ✅ | `cat zz get/set ZZID` |
| `ZZIF` | extended Xcvr Status | yes | ✅ | `cat zz get/set ZZIF` |
| `ZZIO` | Installed options | yes | ✅ | `cat zz get/set ZZIO` |
| `ZZIS` | extended IF shift | yes | ✅ | `cat zz get/set ZZIS` |
| `ZZIT` | extended Filter Shift | yes | ✅ | `cat zz get/set ZZIT` |
| `ZZIU` | extended Filter Shift Reset | yes | ✅ | `cat zz get/set ZZIU` |
| `ZZIV` | IF2VFO | no | ❌ **not wired** |  |
| `ZZJP` | Playback from slot N | yes | ✅ |  |
| `ZZJQ` | Playback/Record container item slot | yes | ✅ |  |
| `ZZJR` | Recording to slot N | yes | ✅ |  |
| `ZZJS` | Stop Wav Recording or Playback | yes | ✅ | `cat zz get/set ZZJS` |
| `ZZKM` | Sends CWX macro | yes | ✅ |  |
| `ZZKO` | CWX Form Control | yes | ✅ | `cat zz get/set ZZKO` |
| `ZZKS` | CWX CW Speed | yes | ✅ | `cat zz get/set ZZKS` |
| `ZZKY` | CWX Send | yes | ✅ |  |
| `ZZLA` | RX0 Gain | yes | ✅ | `cat zz get/set ZZLA` |
| `ZZLB` | RX0 Stereo Bal | yes | ✅ | `cat zz get/set ZZLB` |
| `ZZLC` | RX1 Gain | yes | ✅ | `cat zz get/set ZZLC` |
| `ZZLD` | RX1 Stereo Bal | yes | ✅ | `cat zz get/set ZZLD` |
| `ZZLE` | RX2 audio level | yes | ✅ | `cat zz get/set ZZLE` |
| `ZZLF` | RX2 Stereo Balance | yes | ✅ | `cat zz get/set ZZLF` |
| `ZZLG` | AutoMute RX1 on VFOB TX | yes | ✅ |  |
| `ZZLH` | AutoMute RX2 on VFOA TX | yes | ✅ |  |
| `ZZLI` | PS-A Enable Button | yes | ✅ | `cat zz get/set ZZLI` |
| `ZZMA` | extended MUT function | yes | ✅ | `cat zz get/set ZZMA` |
| `ZZMB` | RX2 Mute | yes | ✅ | `cat zz get/set ZZMB` |
| `ZZMD` | extended modes | yes | ✅ |  |
| `ZZME` | RX2 DSP Mode | yes | ✅ |  |
| `ZZMF` | set multifunction encoder text | yes | ✅ |  |
| `ZZMG` | extended TX Preamp Gain | yes | ✅ | `cat zz get/set ZZMG` |
| `ZZML` | Mode List | yes | ✅ | `cat zz get/set ZZML` |
| `ZZMN` | Filter Presets | yes | ✅ | `cat zz get/set ZZMN` |
| `ZZMO` | Monitor button | yes | ✅ | `cat zz get/set ZZMO` |
| `ZZMR` | extended RX Meter | yes | ✅ |  |
| `ZZMS` | MultiRX Swap checkbox status | yes | ✅ | `cat zz get/set ZZMS` |
| `ZZMT` | extended TX Meter | yes | ✅ |  |
| `ZZMU` | MultiRX button status | yes | ✅ | `cat zz get/set ZZMU` |
| `ZZMV` | Get Memory Channel Count | yes | ✅ | `cat zz get/set ZZMV` |
| `ZZMW` | Get Memory Channel Name | yes | ✅ | `cat zz get/set ZZMW` |
| `ZZMX` | Memeory List | yes | ❌ **not wired** |  |
| `ZZMY` | Save Memory Channel | yes | ✅ | `cat zz get/set ZZMY` |
| `ZZMZ` | Edit Memory Channel | yes | ✅ |  |
| `ZZNA` | Noise Blanker 1 button status | yes | ✅ | `cat zz get/set ZZNA` |
| `ZZNB` | extended nb2 status | yes | ✅ | `cat zz get/set ZZNB` |
| `ZZNC` | RX2 NB Button | yes | ✅ | `cat zz get/set ZZNC` |
| `ZZND` | RX2 NB2 Button | yes | ✅ | `cat zz get/set ZZND` |
| `ZZNE` | Improved rx1 nr status | yes | ✅ | `cat zz get/set ZZNE` |
| `ZZNF` | Improved rx2 nr status | yes | ✅ | `cat zz get/set ZZNF` |
| `ZZNG` | Rx1 NR4 reduction amount | yes | ✅ | `cat zz get/set ZZNG` |
| `ZZNH` | Rx2 NR4 reduction amount | yes | ✅ | `cat zz get/set ZZNH` |
| `ZZNL` | extended nb1 threshold | yes | ✅ | `cat zz get/set ZZNL` |
| `ZZNM` | extended nb2 threshold | yes | ✅ | `cat zz get/set ZZNM` |
| `ZZNN` | RX1 SNB Button | yes | ✅ | `cat zz get/set ZZNN` |
| `ZZNO` | RX2 SNB Button | yes | ✅ | `cat zz get/set ZZNO` |
| `ZZNR` | extended rx1 nr status | yes | ✅ | `cat zz get/set ZZNR` |
| `ZZNS` | extended rx1 nr2 status | yes | ✅ | `cat zz get/set ZZNS` |
| `ZZNT` | ANF status | yes | ✅ | `cat zz get/set ZZNT` |
| `ZZNU` | RX2 ANF status | yes | ✅ | `cat zz get/set ZZNU` |
| `ZZNV` | extended rx2 nr status | yes | ✅ | `cat zz get/set ZZNV` |
| `ZZNW` | extended rx2 nr2 status | yes | ✅ | `cat zz get/set ZZNW` |
| `ZZOA` | RXAnt1 | yes | ✅ |  |
| `ZZOB` | RXAnt2 | yes | ✅ |  |
| `ZZOC` | TXAnt | yes | ✅ |  |
| `ZZOD` | AntMode | yes | ✅ |  |
| `ZZOE` | RX1ExtAnt | yes | ✅ |  |
| `ZZOF` | TXRelays | yes | ✅ |  |
| `ZZOG` | TXRelayEnable | yes | ✅ |  |
| `ZZOH` | TXRelayDelay | yes | ✅ |  |
| `ZZOJ` | Antenna Lock | yes | ✅ |  |
| `ZZOL` | DigL Click Tune Offset | yes | ✅ | `cat zz get/set ZZOL` |
| `ZZOS` | Offset Direction | yes | ✅ |  |
| `ZZOT` | Repeater Freq Offset | yes | ✅ |  |
| `ZZOU` | DigU Click Tune Offset | yes | ✅ | `cat zz get/set ZZOU` |
| `ZZOV` | ATU Enable Button | yes | ✅ |  |
| `ZZOW` | ATY Bypass Button | yes | ✅ |  |
| `ZZOX` | Aries ATU match state | yes | ✅ |  |
| `ZZOZ` | Aries ATU solution erase response | yes | ✅ |  |
| `ZZPA` | extended Preamp status | yes | ✅ | `cat preamp` |
| `ZZPB` | RX2 Preamp Button | yes | ✅ |  |
| `ZZPC` | Drive Level | yes | ✅ | `cat zz get/set ZZPC` |
| `ZZPD` | Center Display Pan | yes | ✅ | `cat zz get/set ZZPD` |
| `ZZPE` | Display Pan Position | yes | ✅ | `cat zz get/set ZZPE` |
| `ZZPK` | COMP status | no | ❌ **not wired** |  |
| `ZZPL` | extended comp threshold | no | ❌ **not wired** |  |
| `ZZPO` | Display Peak button status | yes | ✅ |  |
| `ZZPS` | extended Power Switch | yes | ✅ | `cat power` |
| `ZZPY` | Disply Zoom | yes | ✅ | `cat zz get/set ZZPY` |
| `ZZPZ` | Display Zoom buttons | yes | ✅ |  |
| `ZZQA` | Quick Play button status | yes | ✅ | `cat quickplay` |
| `ZZQB` | Quick Rec button status | yes | ✅ | `cat quickrec` |
| `ZZQK` | Enable QSK Break In | yes | ✅ | `cat zz get/set ZZQK` |
| `ZZQM` | extended Quick Memory Value | yes | ✅ | `cat zz get/set ZZQM` |
| `ZZQR` | Quick Memory Restore | yes | ✅ | `cat zz get/set ZZQR` |
| `ZZQS` | Quick Memory Save | yes | ✅ | `cat zz get/set ZZQS` |
| `ZZRA` | RTTY Offset Enable A | yes | ✅ | `cat zz get/set ZZRA` |
| `ZZRB` | RTTY Offset Enable B | yes | ✅ | `cat zz get/set ZZRB` |
| `ZZRC` | RIT freq clear | yes | ✅ | `cat zz get/set ZZRC` |
| `ZZRD` | RIT Down | yes | ✅ |  |
| `ZZRF` | extended RIT Value | yes | ✅ | `cat zz get/set ZZRF` |
| `ZZRH` | RTTY DIGH offset freq | yes | ✅ | `cat zz get/set ZZRH` |
| `ZZRL` | RTTY DIGL offset freq | yes | ✅ | `cat zz get/set ZZRL` |
| `ZZRM` | extended TX Meter Output | yes | ✅ | `cat zz get/set ZZRM` |
| `ZZRS` | RX2 Enable | yes | ✅ | `cat zz get/set ZZRS` |
| `ZZRT` | RIT button status | yes | ✅ | `cat rit` |
| `ZZRU` | RIT Up | yes | ✅ |  |
| `ZZRV` | Primary Input Voltage | yes | ✅ | `cat zz get/set ZZRV` |
| `ZZRX` | RX1 Atten | yes | ✅ | `cat atten` |
| `ZZRY` | RX2 Atten | yes | ✅ | `cat zz get/set ZZRY` |
| `ZZSA` | Step Down | yes | ✅ | `cat zz get/set ZZSA` |
| `ZZSB` | Step Up | yes | ✅ | `cat zz get/set ZZSB` |
| `ZZSD` | Tune Step Down | yes | ✅ | `cat zz get/set ZZSD` |
| `ZZSF` | extended set filter | yes | ✅ |  |
| `ZZSG` | Step Down | yes | ✅ | `cat zz get/set ZZSG` |
| `ZZSH` | Step Up | yes | ✅ | `cat zz get/set ZZSH` |
| `ZZSM` | extended S Meter | yes | ✅ | `cat zz get/set ZZSM` |
| `ZZSN` | Radio serial number | yes | ✅ | `cat zz get/set ZZSN` |
| `ZZSO` | extended Squelch status | yes | ✅ | `cat zz get/set ZZSO` |
| `ZZSP` | extended vfo split | yes | ✅ | `cat split` |
| `ZZSQ` | extended Squelch Control | yes | ✅ | `cat zz get/set ZZSQ` |
| `ZZSR` | Spur Reduction | yes | ✅ | `cat zz get/set ZZSR` |
| `ZZSS` | CWX Stop | yes | ✅ | `cat zz get/set ZZSS` |
| `ZZST` | extended step size | yes | ✅ | `cat zz get/set ZZST` |
| `ZZSU` | Tune Step Up | yes | ✅ | `cat zz get/set ZZSU` |
| `ZZSV` | RX2 Squelch Button | yes | ✅ | `cat zz get/set ZZSV` |
| `ZZSW` | Swap VFO A/B TX Buttons | yes | ✅ | `cat zz get/set ZZSW` |
| `ZZSX` | RX2 Squelch Threshold | yes | ✅ | `cat zz get/set ZZSX` |
| `ZZSY` | VFO Sync Button | yes | ✅ | `cat zz get/set ZZSY` |
| `ZZSZ` | Zeros selected VFO to current step size | yes | ✅ |  |
| `ZZTA` | CTCSS Enable | yes | ✅ | `cat zz get/set ZZTA` |
| `ZZTB` | CTCSS Frequency | yes | ✅ |  |
| `ZZTC` | TCI server listening status | yes | ✅ | `cat tciserver` |
| `ZZTF` | Show TX Filter | yes | ✅ | `cat zz get/set ZZTF` |
| `ZZTH` | extended TX Filter High | yes | ✅ | `cat zz get/set ZZTH` |
| `ZZTI` | Transmit Inhibit | yes | ✅ | `cat zz get/set ZZTI` |
| `ZZTL` | extended TX Filter Low | yes | ✅ | `cat zz get/set ZZTL` |
| `ZZTM` | TX AF Monitor Gain | yes | ✅ | `cat zz get/set ZZTM` |
| `ZZTO` | Tune Power | yes | ✅ | `cat zz get/set ZZTO` |
| `ZZTP` | TX ProfileCount | yes | ✅ | `cat zz get/set ZZTP` |
| `ZZTS` | Read F5K Temp Sensor | yes | ✅ | `cat zz get/set ZZTS` |
| `ZZTU` | extended TUN status | yes | ✅ | `cat tune` (**TX**) |
| `ZZTV` | Transmit VFO Freq | yes | ✅ |  |
| `ZZTX` | MOX button | yes | ✅ |  |
| `ZZUA` | XVTR Band Names | yes | ✅ | `cat zz get/set ZZUA` |
| `ZZUP` | External PA button | yes | ✅ | `cat zz get/set ZZUP` |
| `ZZUS` | PS Single Cal activate  | yes | ✅ | `cat zz get/set ZZUS` |
| `ZZUT` | extended Two-tone test status | yes | ✅ | `cat zz get/set ZZUT` |
| `ZZUX` | extended VFOA Lock status | yes | ✅ | `cat zz get/set ZZUX` |
| `ZZUY` | extended VFOB Lock status | yes | ✅ | `cat zz get/set ZZUY` |
| `ZZVA` | VAC Enable | yes | ✅ | `cat zz get/set ZZVA` |
| `ZZVB` | VAC RX Gain | yes | ✅ | `cat zz get/set ZZVB` |
| `ZZVC` | VAC TX Gain | yes | ✅ | `cat zz get/set ZZVC` |
| `ZZVD` | VAC Sample Rate | yes | ✅ |  |
| `ZZVE` | VOX Enable | yes | ✅ | `cat zz get/set ZZVE` |
| `ZZVF` | VAC Stereo | yes | ✅ | `cat zz get/set ZZVF` |
| `ZZVG` | VOX Gain | yes | ✅ | `cat zz get/set ZZVG` |
| `ZZVH` | IQ2VAC | yes | ✅ | `cat zz get/set ZZVH` |
| `ZZVI` | VAC Input Cable | yes | ✅ | `cat zz get/set ZZVI` |
| `ZZVJ` | VAC Use RX2 | yes | ✅ | `cat zz get/set ZZVJ` |
| `ZZVK` | VAC2Enable | yes | ✅ | `cat zz get/set ZZVK` |
| `ZZVL` | extended VFO Lock status | yes | ✅ | `cat zz get/set ZZVL` |
| `ZZVM` | VAC Driver | yes | ✅ | `cat zz get/set ZZVM` |
| `ZZVN` | extended Get Version | yes | ✅ | `cat zz get/set ZZVN` |
| `ZZVO` | VAC Output Cable | yes | ✅ | `cat zz get/set ZZVO` |
| `ZZVP` | VAC1Calibrate | yes | ✅ | `cat zz get/set ZZVP` |
| `ZZVQ` | VAC2Driver | yes | ✅ | `cat zz get/set ZZVQ` |
| `ZZVR` | VAC2InputCable | yes | ✅ | `cat zz get/set ZZVR` |
| `ZZVS` | extended vfo swap | yes | ✅ |  |
| `ZZVT` | VAC2OutputCable | yes | ✅ | `cat zz get/set ZZVT` |
| `ZZVU` | VAC2SampleRate | yes | ✅ |  |
| `ZZVV` | VAC2Stereo | yes | ✅ | `cat zz get/set ZZVV` |
| `ZZVW` | VAC2RXGain | yes | ✅ | `cat zz get/set ZZVW` |
| `ZZVX` | VAC2TXGain | yes | ✅ | `cat zz get/set ZZVX` |
| `ZZVY` | VAC1BufferSize | yes | ✅ |  |
| `ZZVZ` | VAC2BufferSize | yes | ✅ |  |
| `ZZWA` | Mixer Mic Gain | yes | ✅ |  |
| `ZZWB` | Mixer LIne In RCA | yes | ✅ |  |
| `ZZWC` | Mixer Line In Phono | yes | ✅ |  |
| `ZZWD` | Mixer Line In DB9 | yes | ✅ |  |
| `ZZWE` | Mixer Mic Select | yes | ✅ |  |
| `ZZWF` | Mixer Line In RCA Select | yes | ✅ |  |
| `ZZWG` | Mixer Line In Phono Select | yes | ✅ |  |
| `ZZWH` | Mixer Line In DB9 Select | yes | ✅ |  |
| `ZZWJ` | Mixer Input Mute All | yes | ✅ |  |
| `ZZWK` | Mixer Output Internal Spkr | yes | ✅ |  |
| `ZZWL` | Mixer Output Ext Spkr | yes | ✅ |  |
| `ZZWM` | Mixer Output Headphone | yes | ✅ |  |
| `ZZWN` | Mixer Line Out RCA | yes | ✅ |  |
| `ZZWO` | Mixer Output Int Spkr Select | yes | ✅ |  |
| `ZZWP` | Mixer Output Ext Spkr Select | yes | ✅ |  |
| `ZZWQ` | Mixer Output Headphone Select | yes | ✅ |  |
| `ZZWR` | Mixer Output Line Out RCA Select | yes | ✅ |  |
| `ZZWS` | Mixer Output Mute All | yes | ✅ |  |
| `ZZWT` | F1500 Mixer Mic Gain | yes | ✅ |  |
| `ZZWU` | F1500 Mixer FlexWire In Gain | yes | ✅ |  |
| `ZZWV` | F1500 Mixer Phones out Level | yes | ✅ |  |
| `ZZWW` | F1500 Mixer FlexWire Out Level | yes | ✅ |  |
| `ZZXA` | Audio Amp | yes | ✅ | `cat zz get/set ZZXA` |
| `ZZXC` | extended XIT Clear | yes | ✅ | `cat zz get/set ZZXC` |
| `ZZXD` | XIT Down | yes | ✅ |  |
| `ZZXF` | extended XIT Value | yes | ✅ | `cat zz get/set ZZXF` |
| `ZZXH` | VOX Hang time | yes | ✅ | `cat zz get/set ZZXH` |
| `ZZXN` | RX1 Combined Status | yes | ✅ | `cat zz get/set ZZXN` |
| `ZZXO` | RX2 Combined Status | yes | ✅ | `cat zz get/set ZZXO` |
| `ZZXS` | XIT button status | yes | ✅ | `cat xit` |
| `ZZXT` | X2TR | yes | ✅ | `cat zz get/set ZZXT` |
| `ZZXU` | XIT Up | yes | ✅ |  |
| `ZZXV` | VFO Combined Status | yes | ✅ | `cat zz get/set ZZXV` |
| `ZZYA` | VAC2DirectIQ | yes | ✅ | `cat zz get/set ZZYA` |
| `ZZYB` | VAC2Calibrate | yes | ✅ | `cat zz get/set ZZYB` |
| `ZZYC` | FM Mic Gain | yes | ✅ | `cat zz get/set ZZYC` |
| `ZZYR` | RX1/RX2 radio button | yes | ✅ | `cat zz get/set ZZYR` |
| `ZZZA` | ganymede amplifier trip state | yes | ✅ |  |
| `ZZZB` | Zero Beat | yes | ✅ | `cat zz get/set ZZZB` |
| `ZZZD` | front panel VFO encoder Down | yes | ✅ |  |
| `ZZZE` | front panel encoder | yes | ✅ |  |
| `ZZZM` | Get Hardware Model | yes | ✅ | `cat zz get/set ZZZM` |
| `ZZZN` | Enable Quick Split | yes | ✅ | `cat zz get/set ZZZN` |
| `ZZZO` | Enable Quick Split and VFO Split | yes | ✅ | `cat zz get/set ZZZO` |
| `ZZZP` | front panel button press | yes | ✅ |  |
| `ZZZQ` | RX1 Auto AGC compensation | yes | ✅ | `cat zz get/set ZZZQ` |
| `ZZZR` | RX2 Auto AGC compensation | yes | ✅ | `cat zz get/set ZZZR` |
| `ZZZS` | front panel s/w version | yes | ✅ |  |
| `ZZZT` | Zoom To Band Recall/Store | yes | ✅ | `cat zz get/set ZZZT` |
| `ZZZU` | front panel VFO encoder Up | yes | ✅ |  |
| `ZZZV` | Get Software Version String | yes | ✅ | `cat zz get/set ZZZV` |
| `ZZZW` | Get SwapVFOWheels state | yes | ✅ | `cat zz get/set ZZZW` |
| `ZZZZ` | Close Serial Port | yes | ✅ | `cat zz get/set ZZZZ` |

### CAT gotchas found while generating this table

- **A naive `grep 'case "'` over `CATParser.cs` overcounts dispatched codes
  by matching commented-out cases.** The first pass of this table (2026-08-17)
  reported only `ZZIV`/`ZZMX` as undispatched; re-checking with comments
  stripped found three more — `ZZDY`, `ZZPK`, `ZZPL` — whose `case` labels
  in `CATParser.cs` *and* whose handler bodies in `CATCommands.cs` are both
  `//`-commented out entirely. Lesson for next time this file is
  regenerated: strip comments before pattern-matching C# source, don't trust
  a bare `grep`.
- **`ZZMX` (Memory List) and `ZZDY` (DX Level) are both marked
  `<active>true</active>` but have no live `case` in `CATParser.cs`'s
  dispatch switch** — sending either does nothing; Thetis silently ignores
  it rather than erroring. This is the same class of bug `AGENTS.md` already
  documents for `ZZQA`/`ZZQB` (which *have* since been wired — see below) and
  `ZZFD`/`ZZFS` (a claimed-twice code caught by a compile failure): a
  command whose struct/description exists doesn't mean it's reachable.
  Don't wire `thetisctl` support for `ZZMX`/`ZZDY` without first confirming
  (or fixing) this upstream.
- **`ZZIV` (IF2VFO), `ZZPK` (COMP status), and `ZZPL` (extended comp
  threshold) are marked `<active>false</active>` and are, consistently,
  also undispatched** — these three behave as expected, unlike `ZZMX`/`ZZDY`.
- **`ZZQA`/`ZZQB` (Quick Play/Quick Rec) are now both active and dispatched**
  in this checkout, confirming they've been fixed since the historical note
  in `AGENTS.md` and `internal/cat/commands.go`'s doc comments (dated
  2026-07-30) — `thetisctl cat quickplay`/`quickrec` rely on this.
- Every other command with `<active>false</active>` (69 more, mostly legacy
  2-letter codes like `AC`, `AL`, `AM`, `AN`, `AR`, `AS`, `BC`, `BP`, `BY`,
  `CA`, `CG`, …) *is* still dispatched — Thetis routes them to a handler,
  but per the doc comments already in `internal/cat/commands.go`, at least
  some of those handlers are deliberate no-op stubs kept for CAT-client
  compatibility (e.g. legacy `RT`/`XT`/`RA`/`PA`/`GT` in favor of their
  `ZZ`-extended equivalents). `<active>false</active>` is not a reliable
  signal that a command is safe to ignore — only that Thetis's own UI/other
  internals don't currently exercise it; verify against `CATCommands.cs`'s
  actual handler body before relying on one.

## Keeping this file current

Thetis is periodically synced from upstream and CAT/TCI command dispatch
can move (`AGENTS.md`); this file will drift. To regenerate it:

**TCI**: re-read the current `doc/TCI_interface_<latest>.pdf` from
[maksimus1210/TCI](https://github.com/maksimus1210/TCI) and diff its command
table against `internal/tci/control.go`'s wire-command list.

**CAT**: from a checkout of the main Thetis repo,

```bash
CAT="Project Files/Source/Console/CAT"
# every struct-defined code + description + active flag
grep -c '<catstruct code=' "$CAT/CATStructs.xml"
# codes CATParser.cs actually dispatches (excludes the "ZZ" trigger itself)
grep -oP 'case "\K[A-Z0-9]+(?=")' "$CAT/CATParser.cs" | sort -u
```

then diff the two lists (as this file's generation did) to find newly
undispatched-but-defined codes, and re-check `internal/cat/commands.go` /
`cmd/thetisctl/cat_cmd.go` for which codes `thetisctl` now sends.

## See also

- `README.md` — day-to-day command tables, usage examples, TX safety
- `AGENTS.md` — contributor contract, including the requirement to verify
  wire formats against Thetis's own source before changing one
- `.claude/skills/thetis-control/SKILL.md` — the agent-facing skill that
  operates `thetisctl`, including the TX confirmation protocol
