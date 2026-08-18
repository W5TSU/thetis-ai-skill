# TCI & CAT command reference

Full command inventories for the two protocols `thetisctl` speaks, generated
directly from source rather than from memory, so this file can be diffed
against the upstream sources and re-generated when they change. See
`README.md` for the curated, day-to-day command tables and usage examples;
this file is the exhaustive cross-reference behind them.

## Sources

| Protocol | Source | Version / commit |
|---|---|---|
| TCI | Expert Electronics' "Universal transceiver control interface – TCI" spec (`doc/TCI_interface_1.6.pdf`, [maksimus1210/TCI](https://github.com/maksimus1210/TCI)) | v1.6, 2021 |
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
set forms); CAT has no equivalent raw-passthrough escape hatch currently.

## Summary

| | Documented | Implemented in thetisctl |
|---|---|---|
| TCI | 80 commands (v1.0 base + v1.4/v1.5/v1.6 additions) | 18 (~23%), plus 5 Thetis-only extensions not in the spec at all |
| CAT | 429 commands (107 legacy Kenwood-style + 322 `ZZxx` extended) | 22 (~5%) |

## TCI commands

`RO`/`WO`/`RW` = read-only / write-only / read-write, per the spec. "Added"
is the TCI spec version each command first appeared in. The five `cw_*`/
`callsign_send` rows are documented in the spec's prose (the CW-macro
section) rather than in its command table, but are real wire commands like
everything else here.

**Total documented commands: 80; implemented with a typed thetisctl command: 18.**

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
| `DDS` | RW | v1.0 | Tuning of the RX's center frequency (panorama center) |  |
| `IF` | RW | v1.0 | IF filter tuning in panorama bandwidth |  |
| `RIT_ENABLE` | RW | v1.0 | Enable RIT | `tci rit` |
| `MODULATION` | RW | v1.0 | Set mode type | `tci modulation` |
| `RX_ENABLE` | RW | v1.0 | Enable software receivers |  |
| `XIT_ENABLE` | RW | v1.0 | Enable XIT | `tci xit` |
| `SPLIT_ENABLE` | RW | v1.0 | Enable SPLIT mode | `tci split` |
| `RIT_OFFSET` | RW | v1.0 | Tune RIT offset | `tci rit-offset` |
| `XIT_OFFSET` | RW | v1.0 | Tune XIT offset | `tci xit-offset` |
| `RX_CHANNEL_ENABLE` | RW | v1.0 | Enable additional receive channels |  |
| `RX_FILTER_BAND` | RW | v1.0 | Adjust IF filter width | `tci filter` |
| `RX_SMETER` | RW | v1.0 | Signal level (S-Meter) in filter bandwidth |  |
| `CW_MACROS_SPEED` | RW | v1.0 | Set CW speed for macros | `tci cw send --speed` |
| `CW_MACROS_SPEED_UP` | WO | v1.0 | Increase CW speed for macros |  |
| `CW_MACROS_SPEED_DOWN` | WO | v1.0 | Decrease CW speed for macros |  |
| `CW_MACROS_DELAY` | RW | v1.0 | Delay between "turn to TX" and "start of macro transmission" |  |
| `TUNE` | RW | v1.0 | Switch RX/TUNE modes | `tci tune (TX)` |
| `IQ_START` | RW | v1.0 | Start IQ signal output |  |
| `IQ_STOP` | RW | v1.0 | Stop IQ signal output |  |
| `IQ_SAMPLERATE` | RW | v1.0 | Set IQ signal sample rate |  |
| `AUDIO_START` | RW | v1.0 | Start audio stream | `tci rx-audio (internal)` |
| `AUDIO_STOP` | RW | v1.0 | Stop audio stream | `tci rx-audio (internal)` |
| `AUDIO_SAMPLERATE` | RW | v1.0 | Set audio stream sample rate |  |
| `SPOT` | WO | v1.0 | Send spot to display |  |
| `SPOT_DELETE` | WO | v1.0 | Delete a spot |  |
| `SPOT_CLEAR` | WO | v1.0 | Clear all spots |  |
| `PROTOCOL` | RO | v1.0 | TCI protocol version, sent on connect |  |
| `TX_POWER` | RO | v1.0 | Output power level, W |  |
| `TX_SWR` | RO | v1.0 | Transmitter SWR value |  |
| `VOLUME` | RW | v1.0 | Main software volume |  |
| `SQL_ENABLE` | RW | v1.0 | On/off squelch |  |
| `SQL_LEVEL` | RW | v1.0 | Squelch threshold |  |
| `VFO` | RW | v1.0 | Set receiver's tuning frequency | `tci vfo` |
| `APP_FOCUS` | RO | v1.0 | Status of the main app window (in focus or not) |  |
| `SET_IN_FOCUS` | WO | v1.0 | Set the main app window in focus |  |
| `MUTE` | RW | v1.0 | Mute — disable/enable overall volume |  |
| `RX_MUTE` | RW | v1.0 | Mute a certain receiver |  |
| `CTCSS_ENABLE` | RW | v1.4 | Enable/disable CTCSS tones |  |
| `CTCSS_MODE` | RW | v1.4 | Switch CTCSS tone modes |  |
| `CTCSS_RX_TONE` | RW | v1.4 | Set CTCSS tone for a receiver |  |
| `CTCSS_TX_TONE` | RW | v1.4 | Set CTCSS tone for a transmitter |  |
| `CTCSS_LEVEL` | RW | v1.4 | Control CTCSS tone level in NFM mode |  |
| `ECODER_SWITCH_RX` | RW | v1.4 | Switch control over active RX with E-Coder panel |  |
| `ECODER_SWITCH_CHANNEL` | RW | v1.4 | Switch control over active channel with E-Coder panel |  |
| `RX_VOLUME` | RW | v1.4 | Volume control for each channel in software receivers |  |
| `RX_BALANCE` | RW | v1.4 | Volume balance control for each channel in software receivers |  |
| `TRX` | RW | v1.5 | Switch between RX/TX modes | `tci ptt (TX)` |
| `DRIVE` | RW | v1.5 | Control transmitter power output | `tci drive` |
| `TUNE_DRIVE` | RW | v1.5 | Control transmitter power output in TUNE mode |  |
| `RX_SENSORS_ENABLE` | WO | v1.5 | Enable/disable sharing of S-meter readings |  |
| `TX_SENSORS_ENABLE` | WO | v1.5 | Enable/disable sharing of transmitter readings |  |
| `RX_SENSORS` | RO | v1.5 | Shared RX signal level (in filter bandwidth) |  |
| `TX_SENSORS` | RO | v1.5 | Shared TX signal parameters (mic level, RMS/PEAK power, SWR) |  |
| `RX_NB_ENABLE` | RW | v1.6 | Enable/disable Noise Blanker (NB) |  |
| `RX_NB_PARAM` | RW | v1.6 | Adjust Noise Blanker (NB) parameters |  |
| `RX_BIN_ENABLE` | RW | v1.6 | Enable/disable pseudo stereo (BIN) |  |
| `RX_NR_ENABLE` | RW | v1.6 | Enable/disable noise reduction (NR) |  |
| `RX_ANC_ENABLE` | RW | v1.6 | Enable/disable Automatic Noise Cancellation (ANC) |  |
| `RX_ANF_ENABLE` | RW | v1.6 | Enable/disable Automatic Notch Filter (ANF) |  |
| `RX_APF_ENABLE` | RW | v1.6 | Enable/disable Analogue Peak Filter (APF) |  |
| `RX_DSE_ENABLE` | RW | v1.6 | Enable/disable Digital Surround Effect (DSE) for CW |  |
| `RX_NF_ENABLE` | RW | v1.6 | Enable/disable band Notch Filters (NF) |  |
| `TX_FREQUENCY` | RO | v1.6 | Transmitter frequency, Hz |  |
| `cw_macros` | WO | v1.0 | Send free-text CW macro (periodic-number + text form) | `tci cw send (TX)` |
| `cw_terminal` | WO | v1.5 | Enable/disable CW terminal mode (stay in TX after macro finishes) |  |
| `cw_msg` | WO | v1.0 | Send CW message with callsign-repeat / mid-transmission-edit support |  |
| `callsign_send` | WO | v1.0 | Edit the callsign of an in-progress cw_msg transmission |  |
| `cw_macros_stop` | WO | v1.0 | Abort in-progress CW macro transmission and unkey | `auto-sent on unkey` |

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

## CAT commands

All 429 commands defined in `CATStructs.xml` (357 marked `<active>true</active>`,
71 `false`), split into the 107 legacy 2-letter (Kenwood TS-2000-compatible)
codes and the 322 `ZZxx` Thetis/FlexRadio-style extended codes. "`<active>`" is that struct's own
flag (Thetis's own marker for whether a command is considered live vs.
reserved/stubbed); "Dispatched" is whether `CATParser.cs`'s dispatch switch
actually routes that code to a handler — generated by diffing
`CATStructs.xml`'s command list against `CATParser.cs`'s `case` labels
directly, not assumed from `<active>`.

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
| `ZZAA` | APF gain | yes | ✅ |  |
| `ZZAB` | APF bandwidth | yes | ✅ |  |
| `ZZAC` | Sets or reads the step size | yes | ✅ |  |
| `ZZAD` | Moves VFOA down nn Tune Steps | yes | ✅ |  |
| `ZZAE` | Moves VFOA down nn Steps | yes | ✅ |  |
| `ZZAF` | Moves VFOA up nn Steps | yes | ✅ |  |
| `ZZAG` | extended AF Gain | yes | ✅ |  |
| `ZZAI` | Auto Info | yes | ✅ |  |
| `ZZAP` | Audio peak filter on off | yes | ✅ |  |
| `ZZAR` | AGCRF | yes | ✅ |  |
| `ZZAS` | RX2 AGCT | yes | ✅ |  |
| `ZZAT` | APF tune | yes | ✅ |  |
| `ZZAU` | Moves VFOA up nn Tune Steps | yes | ✅ |  |
| `ZZAY` | APF type | yes | ✅ |  |
| `ZZBA` | RX2 down one band | yes | ✅ |  |
| `ZZBB` | RX2 up one band | yes | ✅ |  |
| `ZZBD` | Band Down | yes | ✅ |  |
| `ZZBE` | Moves VFOB down nn Steps | yes | ✅ |  |
| `ZZBF` | Moves VFOB up nn Steps | yes | ✅ |  |
| `ZZBG` | extended Band Group | yes | ✅ |  |
| `ZZBI` | extended BIN status | yes | ✅ |  |
| `ZZBM` | Moves VFO B down nn Tune Steps | yes | ✅ |  |
| `ZZBP` | Moves VFO B up nn Tune Steps | yes | ✅ |  |
| `ZZBR` | BCI Reject | yes | ✅ |  |
| `ZZBS` | extended band change | yes | ✅ | `cat band` |
| `ZZBT` | RX2Band | yes | ✅ |  |
| `ZZBU` | Band Up | yes | ✅ |  |
| `ZZBY` | Closes the console | yes | ✅ |  |
| `ZZCB` | Enable Break In | yes | ✅ |  |
| `ZZCD` | CW Break In Delay | yes | ✅ |  |
| `ZZCF` | Show CW Freq | yes | ✅ |  |
| `ZZCI` | CW Iambic Enable | yes | ✅ |  |
| `ZZCL` | extended CW Pitch | yes | ✅ |  |
| `ZZCM` | CW Monitor | yes | ✅ |  |
| `ZZCN` | CTUN Enable | yes | ✅ |  |
| `ZZCO` | RX2 CTUN Enable | yes | ✅ |  |
| `ZZCP` | extended Compander status | yes | ✅ |  |
| `ZZCS` | extended CW Speed | yes | ✅ |  |
| `ZZCT` | CPDR Threshold | yes | ✅ |  |
| `ZZCU` | extended CPU Usage | yes | ✅ |  |
| `ZZDA` | extended Display Average | yes | ✅ |  |
| `ZZDB` | Diversity RX reference | yes | ✅ |  |
| `ZZDC` | Diversity RX2 Gain | yes | ✅ |  |
| `ZZDD` | Diversity Phase | yes | ✅ |  |
| `ZZDE` | Diversity Form Enable | yes | ✅ |  |
| `ZZDF` | CAT Diversity Form | yes | ✅ |  |
| `ZZDG` | Diversity RX1 Gain | yes | ✅ |  |
| `ZZDH` | Diversity RX Source | yes | ✅ |  |
| `ZZDM` | extended Display mode | yes | ✅ |  |
| `ZZDN` | Waterfall Low Level | yes | ✅ |  |
| `ZZDO` | Waterfall High Level | yes | ✅ |  |
| `ZZDP` | Sprectrum Grid Max | yes | ✅ |  |
| `ZZDQ` | Spectrum Grid Min | yes | ✅ |  |
| `ZZDR` | Spectrum Grid Step | yes | ✅ |  |
| `ZZDS` | FreeDV RX decode sync/SNR status | yes | ✅ | `cat freedv (status)` |
| `ZZDT` | RADE RX decoder-input level/clip status | yes | ✅ |  |
| `ZZDU` | DDUTil Status | yes | ✅ |  |
| `ZZDV` | FreeDV RX decode enable status | yes | ✅ | `cat freedv` |
| `ZZDW` | RADE V1 RX decode enable status | yes | ✅ | `cat radae` |
| `ZZDX` | Phone DX button | yes | ✅ |  |
| `ZZDY` | DX Level | yes | ✅ |  |
| `ZZDZ` | RADE RX decode sync/SNR status | yes | ✅ | `cat radae (status)` |
| `ZZEA` | RXEQ Values | yes | ✅ |  |
| `ZZEB` | TXEQ Values | yes | ✅ |  |
| `ZZEM` | Verbose CAT Errors | yes | ✅ |  |
| `ZZER` | RXEQ button status | yes | ✅ |  |
| `ZZET` | TXEQ button status | yes | ✅ |  |
| `ZZFA` | VFO A | yes | ✅ |  |
| `ZZFB` | VFO B | yes | ✅ |  |
| `ZZFD` | FM Deviation | yes | ✅ |  |
| `ZZFH` | DSP Filter High | yes | ✅ |  |
| `ZZFI` | extended current filter name | yes | ✅ |  |
| `ZZFJ` | RX2 DSP Filter | yes | ✅ |  |
| `ZZFL` | DSP Filter Low | yes | ✅ |  |
| `ZZFM` | Flex Model Number | yes | ✅ |  |
| `ZZFR` | RX2 DSP Filter High | yes | ✅ |  |
| `ZZFS` | RX2 DSP Filter Low | yes | ✅ |  |
| `ZZFT` | TX Freq | yes | ✅ |  |
| `ZZFV` | FlexWire single read | yes | ✅ |  |
| `ZZFW` | FlexWire double read | yes | ✅ |  |
| `ZZFX` | FlexWire single | yes | ✅ |  |
| `ZZFY` | FlexWire double | yes | ✅ |  |
| `ZZGA` | Add guid - used by TCPIPcat internally, ignore | yes | ✅ |  |
| `ZZGE` | Noise Gate Enable | yes | ✅ |  |
| `ZZGL` | Noise Gate Level | yes | ✅ |  |
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
| `ZZID` | extended SetRigType | yes | ✅ |  |
| `ZZIF` | extended Xcvr Status | yes | ✅ |  |
| `ZZIO` | Installed options | yes | ✅ |  |
| `ZZIS` | extended IF shift | yes | ✅ |  |
| `ZZIT` | extended Filter Shift | yes | ✅ |  |
| `ZZIU` | extended Filter Shift Reset | yes | ✅ |  |
| `ZZIV` | IF2VFO | no | ❌ **not wired** |  |
| `ZZJP` | Playback from slot N | yes | ✅ |  |
| `ZZJQ` | Playback/Record container item slot | yes | ✅ |  |
| `ZZJR` | Recording to slot N | yes | ✅ |  |
| `ZZJS` | Stop Wav Recording or Playback | yes | ✅ |  |
| `ZZKM` | Sends CWX macro | yes | ✅ |  |
| `ZZKO` | CWX Form Control | yes | ✅ |  |
| `ZZKS` | CWX CW Speed | yes | ✅ |  |
| `ZZKY` | CWX Send | yes | ✅ |  |
| `ZZLA` | RX0 Gain | yes | ✅ |  |
| `ZZLB` | RX0 Stereo Bal | yes | ✅ |  |
| `ZZLC` | RX1 Gain | yes | ✅ |  |
| `ZZLD` | RX1 Stereo Bal | yes | ✅ |  |
| `ZZLE` | RX2 audio level | yes | ✅ |  |
| `ZZLF` | RX2 Stereo Balance | yes | ✅ |  |
| `ZZLG` | AutoMute RX1 on VFOB TX | yes | ✅ |  |
| `ZZLH` | AutoMute RX2 on VFOA TX | yes | ✅ |  |
| `ZZLI` | PS-A Enable Button | yes | ✅ |  |
| `ZZMA` | extended MUT function | yes | ✅ |  |
| `ZZMB` | RX2 Mute | yes | ✅ |  |
| `ZZMD` | extended modes | yes | ✅ |  |
| `ZZME` | RX2 DSP Mode | yes | ✅ |  |
| `ZZMF` | set multifunction encoder text | yes | ✅ |  |
| `ZZMG` | extended TX Preamp Gain | yes | ✅ |  |
| `ZZML` | Mode List | yes | ✅ |  |
| `ZZMN` | Filter Presets | yes | ✅ |  |
| `ZZMO` | Monitor button | yes | ✅ |  |
| `ZZMR` | extended RX Meter | yes | ✅ |  |
| `ZZMS` | MultiRX Swap checkbox status | yes | ✅ |  |
| `ZZMT` | extended TX Meter | yes | ✅ |  |
| `ZZMU` | MultiRX button status | yes | ✅ |  |
| `ZZMV` | Get Memory Channel Count | yes | ✅ |  |
| `ZZMW` | Get Memory Channel Name | yes | ✅ |  |
| `ZZMX` | Memeory List | yes | ❌ **not wired** |  |
| `ZZMY` | Save Memory Channel | yes | ✅ |  |
| `ZZMZ` | Edit Memory Channel | yes | ✅ |  |
| `ZZNA` | Noise Blanker 1 button status | yes | ✅ |  |
| `ZZNB` | extended nb2 status | yes | ✅ |  |
| `ZZNC` | RX2 NB Button | yes | ✅ |  |
| `ZZND` | RX2 NB2 Button | yes | ✅ |  |
| `ZZNE` | Improved rx1 nr status | yes | ✅ |  |
| `ZZNF` | Improved rx2 nr status | yes | ✅ |  |
| `ZZNG` | Rx1 NR4 reduction amount | yes | ✅ |  |
| `ZZNH` | Rx2 NR4 reduction amount | yes | ✅ |  |
| `ZZNL` | extended nb1 threshold | yes | ✅ |  |
| `ZZNM` | extended nb2 threshold | yes | ✅ |  |
| `ZZNN` | RX1 SNB Button | yes | ✅ |  |
| `ZZNO` | RX2 SNB Button | yes | ✅ |  |
| `ZZNR` | extended rx1 nr status | yes | ✅ |  |
| `ZZNS` | extended rx1 nr2 status | yes | ✅ |  |
| `ZZNT` | ANF status | yes | ✅ |  |
| `ZZNU` | RX2 ANF status | yes | ✅ |  |
| `ZZNV` | extended rx2 nr status | yes | ✅ |  |
| `ZZNW` | extended rx2 nr2 status | yes | ✅ |  |
| `ZZOA` | RXAnt1 | yes | ✅ |  |
| `ZZOB` | RXAnt2 | yes | ✅ |  |
| `ZZOC` | TXAnt | yes | ✅ |  |
| `ZZOD` | AntMode | yes | ✅ |  |
| `ZZOE` | RX1ExtAnt | yes | ✅ |  |
| `ZZOF` | TXRelays | yes | ✅ |  |
| `ZZOG` | TXRelayEnable | yes | ✅ |  |
| `ZZOH` | TXRelayDelay | yes | ✅ |  |
| `ZZOJ` | Antenna Lock | yes | ✅ |  |
| `ZZOL` | DigL Click Tune Offset | yes | ✅ |  |
| `ZZOS` | Offset Direction | yes | ✅ |  |
| `ZZOT` | Repeater Freq Offset | yes | ✅ |  |
| `ZZOU` | DigU Click Tune Offset | yes | ✅ |  |
| `ZZOV` | ATU Enable Button | yes | ✅ |  |
| `ZZOW` | ATY Bypass Button | yes | ✅ |  |
| `ZZOX` | Aries ATU match state | yes | ✅ |  |
| `ZZOZ` | Aries ATU solution erase response | yes | ✅ |  |
| `ZZPA` | extended Preamp status | yes | ✅ | `cat preamp` |
| `ZZPB` | RX2 Preamp Button | yes | ✅ |  |
| `ZZPC` | Drive Level | yes | ✅ |  |
| `ZZPD` | Center Display Pan | yes | ✅ |  |
| `ZZPE` | Display Pan Position | yes | ✅ |  |
| `ZZPK` | COMP status | no | ✅ |  |
| `ZZPL` | extended comp threshold | no | ✅ |  |
| `ZZPO` | Display Peak button status | yes | ✅ |  |
| `ZZPS` | extended Power Switch | yes | ✅ | `cat power` |
| `ZZPY` | Disply Zoom | yes | ✅ |  |
| `ZZPZ` | Display Zoom buttons | yes | ✅ |  |
| `ZZQA` | Quick Play button status | yes | ✅ | `cat quickplay` |
| `ZZQB` | Quick Rec button status | yes | ✅ | `cat quickrec` |
| `ZZQK` | Enable QSK Break In | yes | ✅ |  |
| `ZZQM` | extended Quick Memory Value | yes | ✅ |  |
| `ZZQR` | Quick Memory Restore | yes | ✅ |  |
| `ZZQS` | Quick Memory Save | yes | ✅ |  |
| `ZZRA` | RTTY Offset Enable A | yes | ✅ |  |
| `ZZRB` | RTTY Offset Enable B | yes | ✅ |  |
| `ZZRC` | RIT freq clear | yes | ✅ |  |
| `ZZRD` | RIT Down | yes | ✅ |  |
| `ZZRF` | extended RIT Value | yes | ✅ |  |
| `ZZRH` | RTTY DIGH offset freq | yes | ✅ |  |
| `ZZRL` | RTTY DIGL offset freq | yes | ✅ |  |
| `ZZRM` | extended TX Meter Output | yes | ✅ |  |
| `ZZRS` | RX2 Enable | yes | ✅ |  |
| `ZZRT` | RIT button status | yes | ✅ | `cat rit` |
| `ZZRU` | RIT Up | yes | ✅ |  |
| `ZZRV` | Primary Input Voltage | yes | ✅ |  |
| `ZZRX` | RX1 Atten | yes | ✅ | `cat atten` |
| `ZZRY` | RX2 Atten | yes | ✅ |  |
| `ZZSA` | Step Down | yes | ✅ |  |
| `ZZSB` | Step Up | yes | ✅ |  |
| `ZZSD` | Tune Step Down | yes | ✅ |  |
| `ZZSF` | extended set filter | yes | ✅ |  |
| `ZZSG` | Step Down | yes | ✅ |  |
| `ZZSH` | Step Up | yes | ✅ |  |
| `ZZSM` | extended S Meter | yes | ✅ |  |
| `ZZSN` | Radio serial number | yes | ✅ |  |
| `ZZSO` | extended Squelch status | yes | ✅ |  |
| `ZZSP` | extended vfo split | yes | ✅ | `cat split` |
| `ZZSQ` | extended Squelch Control | yes | ✅ |  |
| `ZZSR` | Spur Reduction | yes | ✅ |  |
| `ZZSS` | CWX Stop | yes | ✅ |  |
| `ZZST` | extended step size | yes | ✅ |  |
| `ZZSU` | Tune Step Up | yes | ✅ |  |
| `ZZSV` | RX2 Squelch Button | yes | ✅ |  |
| `ZZSW` | Swap VFO A/B TX Buttons | yes | ✅ |  |
| `ZZSX` | RX2 Squelch Threshold | yes | ✅ |  |
| `ZZSY` | VFO Sync Button | yes | ✅ |  |
| `ZZSZ` | Zeros selected VFO to current step size | yes | ✅ |  |
| `ZZTA` | CTCSS Enable | yes | ✅ |  |
| `ZZTB` | CTCSS Frequency | yes | ✅ |  |
| `ZZTC` | TCI server listening status | yes | ✅ | `cat tciserver` |
| `ZZTF` | Show TX Filter | yes | ✅ |  |
| `ZZTH` | extended TX Filter High | yes | ✅ |  |
| `ZZTI` | Transmit Inhibit | yes | ✅ |  |
| `ZZTL` | extended TX Filter Low | yes | ✅ |  |
| `ZZTM` | TX AF Monitor Gain | yes | ✅ |  |
| `ZZTO` | Tune Power | yes | ✅ |  |
| `ZZTP` | TX ProfileCount | yes | ✅ |  |
| `ZZTS` | Read F5K Temp Sensor | yes | ✅ |  |
| `ZZTU` | extended TUN status | yes | ✅ |  |
| `ZZTV` | Transmit VFO Freq | yes | ✅ |  |
| `ZZTX` | MOX button | yes | ✅ |  |
| `ZZUA` | XVTR Band Names | yes | ✅ |  |
| `ZZUP` | External PA button | yes | ✅ |  |
| `ZZUS` | PS Single Cal activate  | yes | ✅ |  |
| `ZZUT` | extended Two-tone test status | yes | ✅ |  |
| `ZZUX` | extended VFOA Lock status | yes | ✅ |  |
| `ZZUY` | extended VFOB Lock status | yes | ✅ |  |
| `ZZVA` | VAC Enable | yes | ✅ |  |
| `ZZVB` | VAC RX Gain | yes | ✅ |  |
| `ZZVC` | VAC TX Gain | yes | ✅ |  |
| `ZZVD` | VAC Sample Rate | yes | ✅ |  |
| `ZZVE` | VOX Enable | yes | ✅ |  |
| `ZZVF` | VAC Stereo | yes | ✅ |  |
| `ZZVG` | VOX Gain | yes | ✅ |  |
| `ZZVH` | IQ2VAC | yes | ✅ |  |
| `ZZVI` | VAC Input Cable | yes | ✅ |  |
| `ZZVJ` | VAC Use RX2 | yes | ✅ |  |
| `ZZVK` | VAC2Enable | yes | ✅ |  |
| `ZZVL` | extended VFO Lock status | yes | ✅ |  |
| `ZZVM` | VAC Driver | yes | ✅ |  |
| `ZZVN` | extended Get Version | yes | ✅ |  |
| `ZZVO` | VAC Output Cable | yes | ✅ |  |
| `ZZVP` | VAC1Calibrate | yes | ✅ |  |
| `ZZVQ` | VAC2Driver | yes | ✅ |  |
| `ZZVR` | VAC2InputCable | yes | ✅ |  |
| `ZZVS` | extended vfo swap | yes | ✅ |  |
| `ZZVT` | VAC2OutputCable | yes | ✅ |  |
| `ZZVU` | VAC2SampleRate | yes | ✅ |  |
| `ZZVV` | VAC2Stereo | yes | ✅ |  |
| `ZZVW` | VAC2RXGain | yes | ✅ |  |
| `ZZVX` | VAC2TXGain | yes | ✅ |  |
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
| `ZZXA` | Audio Amp | yes | ✅ |  |
| `ZZXC` | extended XIT Clear | yes | ✅ |  |
| `ZZXD` | XIT Down | yes | ✅ |  |
| `ZZXF` | extended XIT Value | yes | ✅ |  |
| `ZZXH` | VOX Hang time | yes | ✅ |  |
| `ZZXN` | RX1 Combined Status | yes | ✅ |  |
| `ZZXO` | RX2 Combined Status | yes | ✅ |  |
| `ZZXS` | XIT button status | yes | ✅ | `cat xit` |
| `ZZXT` | X2TR | yes | ✅ |  |
| `ZZXU` | XIT Up | yes | ✅ |  |
| `ZZXV` | VFO Combined Status | yes | ✅ |  |
| `ZZYA` | VAC2DirectIQ | yes | ✅ |  |
| `ZZYB` | VAC2Calibrate | yes | ✅ |  |
| `ZZYC` | FM Mic Gain | yes | ✅ |  |
| `ZZYR` | RX1/RX2 radio button | yes | ✅ |  |
| `ZZZA` | ganymede amplifier trip state | yes | ✅ |  |
| `ZZZB` | Zero Beat | yes | ✅ |  |
| `ZZZD` | front panel VFO encoder Down | yes | ✅ |  |
| `ZZZE` | front panel encoder | yes | ✅ |  |
| `ZZZM` | Get Hardware Model | yes | ✅ |  |
| `ZZZN` | Enable Quick Split | yes | ✅ |  |
| `ZZZO` | Enable Quick Split and VFO Split | yes | ✅ |  |
| `ZZZP` | front panel button press | yes | ✅ |  |
| `ZZZQ` | RX1 Auto AGC compensation | yes | ✅ |  |
| `ZZZR` | RX2 Auto AGC compensation | yes | ✅ |  |
| `ZZZS` | front panel s/w version | yes | ✅ |  |
| `ZZZT` | Zoom To Band Recall/Store | yes | ✅ |  |
| `ZZZU` | front panel VFO encoder Up | yes | ✅ |  |
| `ZZZV` | Get Software Version String | yes | ✅ |  |
| `ZZZW` | Get SwapVFOWheels state | yes | ✅ |  |
| `ZZZZ` | Close Serial Port | yes | ✅ |  |

### CAT gotchas found while generating this table

- **`ZZMX` (Memory List) is marked `<active>true</active>` but has no `case`
  in `CATParser.cs`'s dispatch switch** — sending it does nothing; Thetis
  silently ignores it rather than erroring. This is the same class of bug
  `AGENTS.md` already documents for `ZZQA`/`ZZQB` (which *have* since been
  wired — see below) and `ZZFD`/`ZZFS` (a claimed-twice code caught by a
  compile failure): a command whose struct/description exists doesn't mean
  it's reachable. Don't wire `thetisctl` support for `ZZMX` without first
  confirming (or fixing) this upstream.
- **`ZZIV` (IF2VFO) is marked `<active>false</active>` and is, consistently,
  also undispatched** — this one behaves as expected, unlike `ZZMX`.
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
