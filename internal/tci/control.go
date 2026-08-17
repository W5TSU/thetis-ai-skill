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
	return c.SendCmd("cw_macros", itoa(rx), encodeCWText(text))
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

// encodeCWText escapes the wire protocol's own delimiter characters out of
// free-text CW message content, matching decodeTciText's inverse mapping
// (TCIServer.cs:8647-8651): ':' -> '^', ',' -> '~', ';' -> '*'. Without this,
// e.g. a comma in the message would be misread as an argument separator.
func encodeCWText(text string) string {
	r := strings.NewReplacer(":", "^", ",", "~", ";", "*")
	return r.Replace(text)
}

func itoa(n int) string { return strconv.Itoa(n) }
