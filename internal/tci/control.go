package tci

import (
	"fmt"
	"strconv"
	"strings"
)

// Typed helpers for the TCI text control commands, wire formats confirmed
// directly against Project Files/Source/Console/TCIServer.cs handler bodies.
// rx selects the receiver: 0 = RX1, 1 = RX2.

// SetVFOFreqHz sets a VFO's frequency in Hz. chan 0 = VFO A, 1 = VFO B.
// Wire: "vfo:<rx>,<chan>,<hz>;" (handleVFOMessage, TCIServer.cs:3859-3907).
func (c *Client) SetVFOFreqHz(rx, ch int, hz int64) error {
	return c.SendCmd("vfo", itoa(rx), itoa(ch), strconv.FormatInt(hz, 10))
}

// tciModes mirrors handleModulationMessage's lowercase mode strings
// (TCIServer.cs:3972-4046).
var tciModes = map[string]bool{
	"lsb": true, "usb": true, "dsb": true, "am": true, "sam": true,
	"fm": true, "nfm": true, "cw": true, "cwl": true, "cwu": true,
	"digl": true, "digu": true,
}

// SetModulation sets the demod mode by TCI's lowercase name: lsb, usb, dsb,
// am, sam, fm, cw, cwl, cwu, digl, digu.
func (c *Client) SetModulation(rx int, mode string) error {
	m := strings.ToLower(mode)
	if !tciModes[m] {
		return fmt.Errorf("tci: unknown modulation %q", mode)
	}
	return c.SendCmd("modulation", itoa(rx), m)
}

// SetSplitEnable enables/disables VFO split.
// Wire: "split_enable:<rx>,<true|false>;" (handleSplitEnableMessage,
// TCIServer.cs:3241-3277).
func (c *Client) SetSplitEnable(rx int, on bool) error {
	return c.SendCmd("split_enable", itoa(rx), strconv.FormatBool(on))
}

// SetRITEnable enables/disables RIT.
// Wire: "rit_enable:<rx>,<true|false>;" (TCIServer.cs:3278-3293).
func (c *Client) SetRITEnable(rx int, on bool) error {
	return c.SendCmd("rit_enable", itoa(rx), strconv.FormatBool(on))
}

// SetXITEnable enables/disables XIT.
// Wire: "xit_enable:<rx>,<true|false>;" (TCIServer.cs:3294-3309).
func (c *Client) SetXITEnable(rx int, on bool) error {
	return c.SendCmd("xit_enable", itoa(rx), strconv.FormatBool(on))
}

// SetRITOffsetHz sets the RIT offset in Hz.
// Wire: "rit_offset:<rx>,<hz>;" (TCIServer.cs:3310-3325).
func (c *Client) SetRITOffsetHz(rx, hz int) error {
	return c.SendCmd("rit_offset", itoa(rx), itoa(hz))
}

// SetXITOffsetHz sets the XIT offset in Hz.
// Wire: "xit_offset:<rx>,<hz>;" (TCIServer.cs:3326-3340).
func (c *Client) SetXITOffsetHz(rx, hz int) error {
	return c.SendCmd("xit_offset", itoa(rx), itoa(hz))
}

// SetFilterBand sets the RX filter passband edges in Hz (lowHz may be
// negative, e.g. CW filters).
// Wire: "rx_filter_band:<rx>,<lowHz>,<highHz>;" (handleRxFilterBand,
// TCIServer.cs:4529-4575).
func (c *Client) SetFilterBand(rx, lowHz, highHz int) error {
	return c.SendCmd("rx_filter_band", itoa(rx), itoa(lowHz), itoa(highHz))
}

// SetStepAttenuatorDB sets the RX step attenuator in dB (>= 0).
// Wire: "rx_step_att_ex:<rx>,<db>;" (handleRxStepAttEx, TCIServer.cs:4905-4926).
func (c *Client) SetStepAttenuatorDB(rx, db int) error {
	if db < 0 {
		return fmt.Errorf("tci: attenuator %d dB must be >= 0", db)
	}
	return c.SendCmd("rx_step_att_ex", itoa(rx), itoa(db))
}

// SetPreampAttenuatorDB sets the RX preamp gain expressed as a
// non-positive attenuation in dB (0 = off, negative = that many dB of gain).
// Wire: "rx_preamp_att_ex:<rx>,<db<=0>;" (handleRxPreampAttEx,
// TCIServer.cs:4927-4944).
func (c *Client) SetPreampAttenuatorDB(rx, db int) error {
	if db > 0 {
		return fmt.Errorf("tci: preamp attenuation %d dB must be <= 0", db)
	}
	return c.SendCmd("rx_preamp_att_ex", itoa(rx), itoa(db))
}

// tciAGCModes mirrors tciModeToAgcMode (TCIServer.cs:2280-2303).
var tciAGCModes = map[string]bool{
	"off": true, "fixed": true, "fixd": true, "long": true, "slow": true,
	"fast": true, "custom": true, "normal": true, "med": true, "medium": true,
}

// SetAGCMode sets the AGC mode: off/fixed, long, slow, medium (=normal), fast, custom.
// Wire: "agc_mode:<rx>,<mode>;" (handleAgcMode, TCIServer.cs:4996-5010).
func (c *Client) SetAGCMode(rx int, mode string) error {
	m := strings.ToLower(mode)
	if !tciAGCModes[m] {
		return fmt.Errorf("tci: unknown agc mode %q", mode)
	}
	return c.SendCmd("agc_mode", itoa(rx), m)
}

// SetAGCGain sets the AGC threshold/gain, clamped server-side to [-20, 120].
// Wire: "agc_gain:<rx>,<gain>;" (handleAgcGain, TCIServer.cs:5011-5027).
func (c *Client) SetAGCGain(rx, gain int) error {
	return c.SendCmd("agc_gain", itoa(rx), itoa(gain))
}

// SetDrive sets TX drive power (0-100).
// Wire: "drive:<rx>,<pwr>;" (handleDrive, TCIServer.cs:4138-4164).
func (c *Client) SetDrive(rx, pwr int) error {
	if pwr < 0 || pwr > 100 {
		return fmt.Errorf("tci: drive %d out of range 0-100", pwr)
	}
	return c.SendCmd("drive", itoa(rx), itoa(pwr))
}

// SetTune keys (true) or unkeys (false) Thetis's TUNE (carrier) mode. This
// transmits RF just like PTT — callers (the thetisctl CLI) must apply the TX
// confirmation gate (internal/safety) before calling this.
// Wire: "tune:<rx>,<true|false>;" (handleTune, TCIServer.cs:4506-4527).
func (c *Client) SetTune(rx int, on bool) error {
	return c.SendCmd("tune", itoa(rx), strconv.FormatBool(on))
}

// SetTrx keys (true) or unkeys (false) PTT for the given receiver. This is a
// raw wire action with no safety gating of its own — callers must apply the
// TX confirmation gate (internal/safety) before calling this.
// Wire: "trx:<rx>,<true|false>;" (handleTrxMessage, TCIServer.cs:3594-3694).
func (c *Client) SetTrx(rx int, on bool) error {
	return c.SendCmd("trx", itoa(rx), strconv.FormatBool(on))
}

// SetTrxTCIAudio keys (true) or unkeys (false) PTT for the given receiver AND
// declares that this connection will supply TX_AUDIO_STREAM frames as the
// modulation source (the 3-arg "tci" form) — required before streaming TX
// audio for it to actually reach the transmitter. Raw wire action, no safety
// gating of its own; callers must apply the TX confirmation gate first.
// Wire: "trx:<rx>,<true|false>,tci;" (TCIServer.cs:3594-3694, args[2]=="tci").
func (c *Client) SetTrxTCIAudio(rx int, on bool) error {
	return c.SendCmd("trx", itoa(rx), strconv.FormatBool(on), "tci")
}

// StartAudio subscribes this connection to RX_AUDIO_STREAM binary frames for rx.
// Wire: "audio_start:<rx>;" (handleAudioStart, TCIServer.cs:6296-6311).
func (c *Client) StartAudio(rx int) error {
	return c.SendCmd("audio_start", itoa(rx))
}

// StopAudio unsubscribes from RX_AUDIO_STREAM frames for rx.
// Wire: "audio_stop:<rx>;".
func (c *Client) StopAudio(rx int) error {
	return c.SendCmd("audio_stop", itoa(rx))
}

// SetAudioSampleType negotiates the sample encoding used for both RX and TX
// audio stream frames on this connection: int16, int24, int32, float32
// (default float32 if never sent).
// Wire: "audio_stream_sample_type:<type>;" (handleAudioStreamSampleType,
// TCIServer.cs:6313-6338).
func (c *Client) SetAudioSampleType(t SampleType) error {
	return c.SendCmd("audio_stream_sample_type", t.WireName())
}

// SetCWMacroSpeedWPM sets the CW macro/message keyer speed in words per
// minute (clamped server-side to 1-99, clampMacroSpeed TCIServer.cs:8642-8645).
// Wire: "cw_macros_speed:<wpm>;" (handleCwMacrosSpeed, TCIServer.cs:3493-3504).
func (c *Client) SetCWMacroSpeedWPM(wpm int) error {
	return c.SendCmd("cw_macros_speed", itoa(wpm))
}

// SendCWMacro sends free text to be keyed as CW by Thetis's own macro
// engine (TCICWController.SendMacro, TCIServer.cs:8449-8462), which manages
// PTT/MOX itself — callers do NOT need to (and should not) call SetTrx
// around this. The engine sends an unsolicited "cw_macros_empty:<rx>;" text
// frame when the message finishes keying (TCIServer.cs:1991-1993). This is a
// raw wire action with no safety gating of its own — callers must apply the
// TX confirmation gate (internal/safety) before calling this.
// Wire: "cw_macros:<rx>,<text>;" (handleCwMacros, TCIServer.cs:3541-3549).
func (c *Client) SendCWMacro(rx int, text string) error {
	return c.SendCmd("cw_macros", itoa(rx), escapeTCIText(text))
}

// StopCWMacros aborts any in-progress CW macro transmission owned by this
// connection and unkeys. Sent as a bare command with no colon — see
// SendBareCmd's doc comment.
// Wire: "cw_macros_stop;" (handleCwMacrosStop, TCIServer.cs:5577-5579).
func (c *Client) StopCWMacros() error {
	return c.SendBareCmd("cw_macros_stop")
}

// SetPower turns Thetis's radio engine on/off — the same action as the main
// Power button (console.PowerOn / chkPower), starting or stopping the HPSDR
// hardware connection and DSP audio engine. It does NOT touch mains power to
// the radio hardware itself. Sent as a bare command with no colon — see
// SendBareCmd's doc comment. Thetis broadcasts the same "start;"/"stop;"
// frame to every connected TCI client (including this one) once the state
// actually changes (PowerChange → sendStart/sendStop, TCIServer.cs:1500-1504,
// 1911-1917) — callers that need confirmation should watch for that frame
// via RecvCmd rather than assume this call was synchronous.
// Wire: "start;" (on) / "stop;" (off) (handleStart/handleStop, TCIServer.cs:3227-3236).
func (c *Client) SetPower(on bool) error {
	if on {
		return c.SendBareCmd("start")
	}
	return c.SendBareCmd("stop")
}

// The functions below extend coverage to the rest of the commands
// Thetis's TCIServer.cs actually implements from the wider TCI v1.6 spec
// (see PROTOCOLS.md). A meaningful chunk of the spec's remaining commands —
// VFO_LIMITS, IF_LIMITS, TRX_COUNT, CHANNEL_COUNT, DEVICE, RECEIVE_ONLY,
// MODULATIONS_LIST, TX_ENABLE, READY, TX_FOOTSWITCH, PROTOCOL, TX_POWER,
// TX_SWR, APP_FOCUS, RX_SMETER, RX_SENSORS, TX_SENSORS, RX_NB_PARAM,
// CTCSS_* (5), ECODER_SWITCH_* (2), RX_ANC_ENABLE, RX_DSE_ENABLE, and
// callsign_send — do NOT exist anywhere in this Thetis checkout's
// TCIServer.cs (grepped, zero occurrences of any of their wire tokens) and
// are deliberately NOT implemented here; sending them would be a silent
// no-op the server ignores. TX_FREQUENCY exists only as a server-push
// broadcast with no client-request form, so it's already fully reachable
// via `tci query tx_frequency` (see README.md/SKILL.md) without a typed
// wrapper. See PROTOCOLS.md for the full breakdown.

// SetDDSFreqHz retunes the RX's panorama/DDS center frequency in Hz — moves
// the whole panadapter, dragging the VFO along with it to preserve the same
// IF offset (distinct from SetVFOFreqHz, which moves the VFO alone within a
// fixed panorama). Wire: "dds:<rx>,<hz>;" (handleDDS, TCIServer.cs:3808-3852).
func (c *Client) SetDDSFreqHz(rx int, hz int64) error {
	return c.SendCmd("dds", itoa(rx), strconv.FormatInt(hz, 10))
}

// SetIFOffsetHz sets a VFO's frequency indirectly, as an offset in Hz from
// the current DDS/panorama center (chan 0 = VFO A, 1 = VFO B) — the
// spec-native way to reposition a VFO without moving the panorama; most
// callers want the simpler SetVFOFreqHz instead. Wire:
// "if:<rx>,<chan>,<offsetHz>;" (handleIF, TCIServer.cs:3710-3806).
func (c *Client) SetIFOffsetHz(rx, chanNum, offsetHz int) error {
	return c.SendCmd("if", itoa(rx), itoa(chanNum), itoa(offsetHz))
}

// SetRXEnable enables/disables RX2 as a software receiver (rx must be 1;
// RX1/rx 0 is always enabled and this cannot disable it — TCIServer.cs
// silently no-ops that case). Wire: "rx_enable:<rx>,<true|false>;"
// (handleRXEnable, TCIServer.cs:4595-4629).
func (c *Client) SetRXEnable(rx int, on bool) error {
	return c.SendCmd("rx_enable", itoa(rx), strconv.FormatBool(on))
}

// SetRXChannelEnable enables/disables an additional receive channel (VFOA
// sub-receiver when rx=0,chan=1; has no effect for rx=1 since RX2 has no
// sub-receiver — TCIServer.cs always reports chan=1 disabled there). Wire:
// "rx_channel_enable:<rx>,<chan>,<true|false>;" (handleRxChannelEnable,
// TCIServer.cs:6252-6280).
func (c *Client) SetRXChannelEnable(rx, chanNum int, on bool) error {
	return c.SendCmd("rx_channel_enable", itoa(rx), itoa(chanNum), strconv.FormatBool(on))
}

// IncreaseCWMacroSpeedWPM bumps the CW macro/message keyer speed up by amount
// WPM (clamped server-side same as SetCWMacroSpeedWPM). Write-only — no
// reply. Wire: "cw_macros_speed_up:<amount>;" (handleCwMacrosSpeedUp,
// TCIServer.cs:3529-3534).
func (c *Client) IncreaseCWMacroSpeedWPM(amount int) error {
	return c.SendCmd("cw_macros_speed_up", itoa(amount))
}

// DecreaseCWMacroSpeedWPM bumps the CW macro/message keyer speed down by
// amount WPM. Write-only — no reply. Wire: "cw_macros_speed_down:<amount>;"
// (handleCwMacrosSpeedDown, TCIServer.cs:3535-3540).
func (c *Client) DecreaseCWMacroSpeedWPM(amount int) error {
	return c.SendCmd("cw_macros_speed_down", itoa(amount))
}

// SetCWMacroDelayMs sets the delay between keying TX and the CW macro/
// message engine starting to send, in milliseconds. Wire:
// "cw_macros_delay:<ms>;" (handleCwMacrosDelay, TCIServer.cs:3505-3516).
func (c *Client) SetCWMacroDelayMs(ms int) error {
	return c.SendCmd("cw_macros_delay", itoa(ms))
}

// StartIQ subscribes this connection to IQ_STREAM binary frames for rx —
// the IQ analogue of StartAudio; frames arrive via RecvAudioFrame with
// StreamType == StreamIQ. Wire: "iq_start:<rx>;" (handleIQStart,
// TCIServer.cs:6202-6218).
func (c *Client) StartIQ(rx int) error {
	return c.SendCmd("iq_start", itoa(rx))
}

// StopIQ unsubscribes from IQ_STREAM frames for rx.
// Wire: "iq_stop:<rx>;".
func (c *Client) StopIQ(rx int) error {
	return c.SendCmd("iq_stop", itoa(rx))
}

// SetIQSampleRateHz negotiates the IQ stream's advertised sample rate. Note:
// as of this Thetis checkout it is cosmetic only — Thetis echoes back
// whatever value is sent without actually changing hardware sample rate
// (handleIQSampleRate's comment: "we dont change Thetis H/W sample rate for
// now"); the real rate delivered in each StreamHeader.SampleRate is whatever
// Thetis's DSP is actually running. Wire: "iq_samplerate:<hz>;"
// (handleIQSampleRate, TCIServer.cs:6110-6127).
func (c *Client) SetIQSampleRateHz(hz int) error {
	return c.SendCmd("iq_samplerate", itoa(hz))
}

// SetAudioSampleRateHz sets the RX audio stream's sample rate in Hz — one of
// 8000/12000/24000/48000; any other value is accepted on the wire but
// silently ignored server-side (handleAudioSampleRate). Wire:
// "audio_samplerate:<hz>;" (TCIServer.cs:6145-6200).
func (c *Client) SetAudioSampleRateHz(hz int) error {
	return c.SendCmd("audio_samplerate", itoa(hz))
}

// SendSpot pushes a spot to Thetis's panadapter/band-map display. mode is a
// DSPMode name (e.g. "usb", "cw" — same set as SetModulation); argb is a
// 32-bit 0xAARRGGBB color (e.g. 0x00FF0000 == 16711680 == red); extra is
// optional free text, escaped the same as CW macro text. Wire:
// "spot:<callsign>,<mode>,<hz>,<argb>[,<extra>];" (handleSpot,
// TCIServer.cs:4339-4595 area).
func (c *Client) SendSpot(callsign, mode string, hz int64, argb uint32, extra string) error {
	args := []string{callsign, mode, strconv.FormatInt(hz, 10), strconv.FormatUint(uint64(argb), 10)}
	if extra != "" {
		args = append(args, escapeTCIText(extra))
	}
	return c.SendCmd("spot", args...)
}

// DeleteSpot removes a previously-sent spot by callsign.
// Wire: "spot_delete:<callsign>;" (handleDeleteSpot, TCIServer.cs:4076-4082).
func (c *Client) DeleteSpot(callsign string) error {
	return c.SendCmd("spot_delete", callsign)
}

// ClearSpots removes every spot from the display. Sent as a bare command
// with no colon — see SendBareCmd's doc comment.
// Wire: "spot_clear;" (handleSpotClear, TCIServer.cs:3237-3240).
func (c *Client) ClearSpots() error {
	return c.SendBareCmd("spot_clear")
}

// SetVolumeDB sets the main RX audio volume in dB, -60 (muted) to 0 (full).
// Wire: "volume:<db>;" (handleVolume, TCIServer.cs:4313-4324).
func (c *Client) SetVolumeDB(db float64) error {
	return c.SendCmd("volume", strconv.FormatFloat(db, 'f', -1, 64))
}

// SetSquelchEnable enables/disables squelch for rx.
// Wire: "sql_enable:<rx>,<true|false>;" (handleSqlEnable,
// TCIServer.cs:3436-3451).
func (c *Client) SetSquelchEnable(rx int, on bool) error {
	return c.SendCmd("sql_enable", itoa(rx), strconv.FormatBool(on))
}

// SetSquelchLevelDB sets the squelch threshold in dB, clamped server-side to
// [-140, 0]. Wire: "sql_level:<rx>,<db>;" (handleSqlLevel,
// TCIServer.cs:3452-3468).
func (c *Client) SetSquelchLevelDB(rx, db int) error {
	return c.SendCmd("sql_level", itoa(rx), itoa(db))
}

// SetInFocus brings Thetis's main window to the foreground. Sent as a bare
// command with no colon — see SendBareCmd's doc comment.
// Wire: "set_in_focus;" (handleSetInFocus, TCIServer.cs:3223-3226).
func (c *Client) SetInFocus() error {
	return c.SendBareCmd("set_in_focus")
}

// SetMute mutes/unmutes overall RX audio (both receivers at once — there is
// no rx argument; use SetRXMute for a single receiver). Wire: "mute:<true|
// false>;" (handleMute, TCIServer.cs:4214-4231).
func (c *Client) SetMute(on bool) error {
	return c.SendCmd("mute", strconv.FormatBool(on))
}

// SetRXMute mutes/unmutes a single receiver's audio.
// Wire: "rx_mute:<rx>,<true|false>;" (handleMuteRX, TCIServer.cs:4232-4255).
func (c *Client) SetRXMute(rx int, on bool) error {
	return c.SendCmd("rx_mute", itoa(rx), strconv.FormatBool(on))
}

// SetRXVolumeDB sets per-channel RX audio gain in dB, -60 to 0 (chan 0 =
// VFOA/main, 1 = VFOA sub-receiver for rx=0; chan is otherwise unused for
// rx=1/RX2). Wire: "rx_volume:<rx>,<chan>,<db>;" (handleRxVolume,
// TCIServer.cs:4794-4855).
func (c *Client) SetRXVolumeDB(rx, chanNum int, db float64) error {
	return c.SendCmd("rx_volume", itoa(rx), itoa(chanNum), strconv.FormatFloat(db, 'f', 2, 64))
}

// SetRXBalance sets per-channel stereo balance, -40 to 40 (negative = more
// left). Wire: "rx_balance:<rx>,<chan>,<balance>;" (handleRxBalance,
// TCIServer.cs:4856-4882).
func (c *Client) SetRXBalance(rx, chanNum int, balance float64) error {
	return c.SendCmd("rx_balance", itoa(rx), itoa(chanNum), strconv.FormatFloat(balance, 'f', 2, 64))
}

// SetTuneDrive sets TX drive power used specifically while in TUNE mode,
// 0-100 — independent of SetDrive's normal-TX drive level (which source
// applies depends on Thetis's Setup "Tune Drive Source" setting). Wire:
// "tune_drive:<rx>,<pwr>;" (handleTuneDrive, TCIServer.cs:4165-4213).
func (c *Client) SetTuneDrive(rx, pwr int) error {
	return c.SendCmd("tune_drive", itoa(rx), itoa(pwr))
}

// SetRXSensorsEnable enables/disables this connection receiving periodic
// unsolicited RX_SENSORS-shaped S-meter broadcasts (rx_sensors, which — like
// RX_SENSORS/TX_SENSORS generally — is a spec command Thetis does NOT
// implement; only the enable/disable toggle itself is wired). intervalMs is
// optional; pass 0 to omit it and use Thetis's current interval. Write-only
// — no reply. Wire: "rx_sensors_enable:<true|false>[,<intervalMs>];"
// (handleRxSensorsEnable, TCIServer.cs:4631-4641).
func (c *Client) SetRXSensorsEnable(on bool, intervalMs int) error {
	if intervalMs > 0 {
		return c.SendCmd("rx_sensors_enable", strconv.FormatBool(on), itoa(intervalMs))
	}
	return c.SendCmd("rx_sensors_enable", strconv.FormatBool(on))
}

// SetTXSensorsEnable is TX_SENSORS_ENABLE's TX-side counterpart to
// SetRXSensorsEnable — see its doc comment for the same TX_SENSORS caveat.
// Wire: "tx_sensors_enable:<true|false>[,<intervalMs>];"
// (handleTxSensorsEnable, TCIServer.cs:4642-4652).
func (c *Client) SetTXSensorsEnable(on bool, intervalMs int) error {
	if intervalMs > 0 {
		return c.SendCmd("tx_sensors_enable", strconv.FormatBool(on), itoa(intervalMs))
	}
	return c.SendCmd("tx_sensors_enable", strconv.FormatBool(on))
}

// SetNoiseBlankerEnable enables/disables the classic noise blanker (NB).
// Wire: "rx_nb_enable:<rx>,<true|false>;" (handleRxNBEnable, TCIServer.cs:
// 4707-4739 — the plain, non-"_ex" form; Thetis's "rx_nb_enable_ex" 3-arg
// NB-select variant is a Thetis extension beyond the v1.6 spec, not wired
// here).
func (c *Client) SetNoiseBlankerEnable(rx int, on bool) error {
	return c.SendCmd("rx_nb_enable", itoa(rx), strconv.FormatBool(on))
}

// SetBinauralEnable enables/disables pseudo-stereo (BIN) RX audio.
// Wire: "rx_bin_enable:<rx>,<true|false>;" (handleRxBinEnable,
// TCIServer.cs:3343-3358).
func (c *Client) SetBinauralEnable(rx int, on bool) error {
	return c.SendCmd("rx_bin_enable", itoa(rx), strconv.FormatBool(on))
}

// SetNoiseReductionEnable enables/disables noise reduction (NR).
// Wire: "rx_nr_enable:<rx>,<true|false>;" (handleNREnable, TCIServer.cs:
// 4654-4706 — the plain, non-"_ex" form; the "_ex" 3-arg variant selecting
// which NR algorithm is a Thetis extension beyond the v1.6 spec, not wired
// here).
func (c *Client) SetNoiseReductionEnable(rx int, on bool) error {
	return c.SendCmd("rx_nr_enable", itoa(rx), strconv.FormatBool(on))
}

// SetAutoNotchEnable enables/disables the automatic notch filter (ANF).
// Wire: "rx_anf_enable:<rx>,<true|false>;" (handleAnfEnable,
// TCIServer.cs:4740-4766).
func (c *Client) SetAutoNotchEnable(rx int, on bool) error {
	return c.SendCmd("rx_anf_enable", itoa(rx), strconv.FormatBool(on))
}

// SetAudioPeakFilterEnable enables/disables the CW audio peak filter (APF).
// Wire: "rx_apf_enable:<rx>,<true|false>;" (handleRxApfEnable,
// TCIServer.cs:3359-3383).
func (c *Client) SetAudioPeakFilterEnable(rx int, on bool) error {
	return c.SendCmd("rx_apf_enable", itoa(rx), strconv.FormatBool(on))
}

// SetNotchFilterEnable enables/disables Thetis's manual/tracking notch
// filter (NF/TNF). Wire: "rx_nf_enable:<rx>,<true|false>;" (handleRxNfEnable,
// TCIServer.cs:3384-3399).
func (c *Client) SetNotchFilterEnable(rx int, on bool) error {
	return c.SendCmd("rx_nf_enable", itoa(rx), strconv.FormatBool(on))
}

// SetCWTerminalEnable enables/disables CW Terminal mode: when on, the radio
// stays keyed in TX after a cw_macros/SendCWMacro message finishes instead
// of auto-unkeying, and Thetis sends an unsolicited "cw_macros_empty:<rx>;"
// frame when the message completes. Configuration only — does not itself
// key the transmitter, but changes what happens when a subsequent CW send
// does; callers gating that subsequent send is what provides TX safety.
// Write-only, always 2 args (no bare/get form). Wire:
// "cw_terminal:<rx>,<true|false>;" (handleCwTerminal, TCIServer.cs:3550-3558).
func (c *Client) SetCWTerminalEnable(rx int, on bool) error {
	return c.SendCmd("cw_terminal", itoa(rx), strconv.FormatBool(on))
}

// SendCWMessage keys CW text via Thetis's macro/message engine with
// callsign-repeat support: prefix is sent before callsign, suffix after
// (e.g. "TU" / "RA6LH" / "599 004"). Pass "_" for prefix to send no text
// before the callsign, matching the spec's documented convention. This
// genuinely transmits RF exactly like SendCWMacro — callers (the thetisctl
// CLI) must apply the TX confirmation gate (internal/safety) before calling
// this. Wire: "cw_msg:<rx>,<prefix>,<callsign>,<suffix>;" (handleCwMsg,
// TCIServer.cs:3559-3577).
func (c *Client) SendCWMessage(rx int, prefix, callsign, suffix string) error {
	return c.SendCmd("cw_msg", itoa(rx), escapeTCIText(prefix), escapeTCIText(callsign), escapeTCIText(suffix))
}

// EditCWMessageCallsign edits the callsign of an in-progress SendCWMessage
// transmission (e.g. after copying more of the other station's call).
// Thetis applies this only to symbols not yet sent; once the callsign
// portion has already gone out, this is silently ignored. Does not itself
// key the transmitter — it only affects an already-keyed message — so it is
// not gated behind --confirm-tx. Wire: "cw_msg:<callsign>;" (handleCwMsg's
// 1-arg form, TCIServer.cs:3563-3567) — this is also where the spec's
// separate "callsign_send" command's functionality lives in Thetis, which
// has no callsign_send wire command of its own (grepped, zero occurrences).
func (c *Client) EditCWMessageCallsign(callsign string) error {
	return c.SendCmd("cw_msg", escapeTCIText(callsign))
}

// escapeTCIText escapes the wire protocol's own delimiter characters out of
// free-text field content (CW macro/message text, spot free text), matching
// decodeTciText's inverse mapping (TCIServer.cs:8647-8651): ':' -> '^',
// ',' -> '~', ';' -> '*'. Without this, e.g. a comma in the text would be
// misread as an argument separator.
func escapeTCIText(text string) string {
	r := strings.NewReplacer(":", "^", ",", "~", ";", "*")
	return r.Replace(text)
}

func itoa(n int) string { return strconv.Itoa(n) }
