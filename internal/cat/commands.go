package cat

import (
	"fmt"
	"strconv"
	"strings"
)

// Mode codes match KString2Mode/Mode2KString in
// Project Files/Source/Console/CAT/CATCommands.cs:9971-10022. Code "8" is
// unused by Thetis; DIGL/DIGU alias to LSB/USB on the wire when the "Dig U is
// USB" console option is enabled, but the plain codes below always round-trip.
var modeToCode = map[string]string{
	"LSB":  "1",
	"USB":  "2",
	"CW":   "3",
	"CWU":  "3",
	"FM":   "4",
	"AM":   "5",
	"DIGL": "6",
	"CWL":  "7",
	"CWR":  "7",
	"DIGU": "9",
}

var codeToMode = map[string]string{
	"1": "LSB",
	"2": "USB",
	"3": "CWU",
	"4": "FM",
	"5": "AM",
	"6": "DIGL",
	"7": "CWL",
	"9": "DIGU",
}

// AGC codes match the AGCMode enum, Project Files/Source/Console/enums.cs:152-162.
var agcToCode = map[string]string{
	"FIXED":  "0",
	"LONG":   "1",
	"SLOW":   "2",
	"MEDIUM": "3",
	"MED":    "3",
	"FAST":   "4",
	"CUSTOM": "5",
}

var codeToAGC = map[string]string{
	"0": "FIXED",
	"1": "LONG",
	"2": "SLOW",
	"3": "MEDIUM",
	"4": "FAST",
	"5": "CUSTOM",
}

// Band codes match Band2String/String2Band in
// Project Files/Source/Console/CAT/CATCommands.cs:10155-10253.
var bandToCode = map[string]string{
	"160": "160", "80": "080", "60": "060", "40": "040", "30": "030",
	"20": "020", "17": "017", "15": "015", "12": "012", "10": "010",
	"6": "006", "2": "002", "GEN": "888", "WWV": "999",
	"V0": "V00", "V1": "V01", "V2": "V02", "V3": "V03", "V4": "V04",
	"V5": "V05", "V6": "V06", "V7": "V07", "V8": "V08", "V9": "V09",
	"V10": "V10", "V11": "V11", "V12": "V12", "V13": "V13",
}

var codeToBand = func() map[string]string {
	m := make(map[string]string, len(bandToCode))
	for name, code := range bandToCode {
		m[code] = name
	}
	return m
}()

func boolDigit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func digitBool(s string) (bool, error) {
	switch strings.TrimSpace(s) {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("cat: expected 0/1, got %q", s)
	}
}

func vfoFreqCode(vfo string) (string, error) {
	switch strings.ToUpper(vfo) {
	case "A":
		return "FA", nil
	case "B":
		return "FB", nil
	default:
		return "", fmt.Errorf("cat: unknown vfo %q (want A or B)", vfo)
	}
}

// SetVFOFreqHz sets VFO A or B frequency in Hz. Wire format: "FA"/"FB" + an
// 11-digit zero-padded Hz value (CATCommands.cs ~193-206, CATStructs.xml FA/FB).
func (c *Client) SetVFOFreqHz(vfo string, hz uint64) error {
	code, err := vfoFreqCode(vfo)
	if err != nil {
		return err
	}
	if hz > 99999999999 {
		return fmt.Errorf("cat: frequency %d Hz exceeds 11-digit wire format", hz)
	}
	return c.Set(code, fmt.Sprintf("%011d", hz))
}

// GetVFOFreqHz reads VFO A or B frequency in Hz.
func (c *Client) GetVFOFreqHz(vfo string) (uint64, error) {
	code, err := vfoFreqCode(vfo)
	if err != nil {
		return 0, err
	}
	reply, err := c.Query(code)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(reply), 10, 64)
}

// SetMode sets the demod mode by name: USB, LSB, CW (=CWU), CWL (=CW-R), FM, AM, DIGU, DIGL.
func (c *Client) SetMode(mode string) error {
	code, ok := modeToCode[strings.ToUpper(mode)]
	if !ok {
		return fmt.Errorf("cat: unknown mode %q", mode)
	}
	return c.Set("MD", code)
}

// GetMode reads the current demod mode name.
func (c *Client) GetMode() (string, error) {
	reply, err := c.Query("MD")
	if err != nil {
		return "", err
	}
	mode, ok := codeToMode[strings.TrimSpace(reply)]
	if !ok {
		return "", fmt.Errorf("cat: unknown mode code %q", reply)
	}
	return mode, nil
}

// SetRIT enables/disables RIT. Wire command ZZRT (CATCommands.cs:6099-6121);
// the legacy Kenwood "RT" command is a disabled stub in this codebase.
func (c *Client) SetRIT(on bool) error {
	return c.Set("ZZRT", boolDigit(on))
}

// GetRIT reads RIT enabled state.
func (c *Client) GetRIT() (bool, error) {
	reply, err := c.Query("ZZRT")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetXIT enables/disables XIT. Wire command ZZXS (CATCommands.cs:8360-8383);
// the legacy Kenwood "XT" command is a disabled stub in this codebase.
func (c *Client) SetXIT(on bool) error {
	return c.Set("ZZXS", boolDigit(on))
}

// GetXIT reads XIT enabled state.
func (c *Client) GetXIT() (bool, error) {
	reply, err := c.Query("ZZXS")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetSplit enables/disables VFO split. Wire command ZZSP (CATCommands.cs:6370-6402).
func (c *Client) SetSplit(on bool) error {
	return c.Set("ZZSP", boolDigit(on))
}

// GetSplit reads VFO split state.
func (c *Client) GetSplit() (bool, error) {
	reply, err := c.Query("ZZSP")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetAGC sets the AGC mode: FIXED, LONG, SLOW, MEDIUM, FAST, CUSTOM. Wire
// command ZZGT (CATCommands.cs:3217-3236); the legacy Kenwood "GT" command
// only supports get in this codebase (delegates to ZZGT internally).
func (c *Client) SetAGC(mode string) error {
	code, ok := agcToCode[strings.ToUpper(mode)]
	if !ok {
		return fmt.Errorf("cat: unknown agc mode %q", mode)
	}
	return c.Set("ZZGT", code)
}

// GetAGC reads the current AGC mode name.
func (c *Client) GetAGC() (string, error) {
	reply, err := c.Query("ZZGT")
	if err != nil {
		return "", err
	}
	mode, ok := codeToAGC[strings.TrimSpace(reply)]
	if !ok {
		return "", fmt.Errorf("cat: unknown agc code %q", reply)
	}
	return mode, nil
}

// SetAttenuatorDB sets the RX1 step attenuator, 0-31 dB. Wire command ZZRX
// (CATCommands.cs:6176-6197); the legacy Kenwood "RA" command is a disabled
// stub in this codebase.
func (c *Client) SetAttenuatorDB(db int) error {
	if db < 0 || db > 31 {
		return fmt.Errorf("cat: attenuator %d dB out of range 0-31", db)
	}
	return c.Set("ZZRX", fmt.Sprintf("%02d", db))
}

// GetAttenuatorDB reads the RX1 step attenuator in dB.
func (c *Client) GetAttenuatorDB() (int, error) {
	reply, err := c.Query("ZZRX")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(reply))
}

// SetPreamp sets the RX1 preamp level, 0-9 (PreampMode enum,
// Project Files/Source/Console/enums.cs:236-251: 0=off 1=on 2=-10dB 3=-20dB
// 4=-30dB 5=-40dB 6=-50dB 7=SA-10dB 8=SA-20dB 9=SA-30dB). Wire command ZZPA
// (CATCommands.cs:5460-...); the legacy Kenwood "PA" command is a disabled
// stub in this codebase.
func (c *Client) SetPreamp(level int) error {
	if level < 0 || level > 9 {
		return fmt.Errorf("cat: preamp level %d out of range 0-9", level)
	}
	return c.Set("ZZPA", strconv.Itoa(level))
}

// SetBand sets the band by name: 160, 80, 60, 40, 30, 20, 17, 15, 12, 10, 6,
// 2, GEN, WWV, V0-V13. Wire command ZZBS (CATCommands.cs:1620-1623, delegates
// to GetBand at 10102-10127).
func (c *Client) SetBand(band string) error {
	code, ok := bandToCode[strings.ToUpper(band)]
	if !ok {
		return fmt.Errorf("cat: unknown band %q", band)
	}
	return c.Set("ZZBS", code)
}

// GetBand reads the current band name.
func (c *Client) GetBand() (string, error) {
	reply, err := c.Query("ZZBS")
	if err != nil {
		return "", err
	}
	band, ok := codeToBand[strings.TrimSpace(reply)]
	if !ok {
		return "", fmt.Errorf("cat: unknown band code %q", reply)
	}
	return band, nil
}

// GetID reads the emulated rig ID (CATCommands.cs:295-317, e.g. "019" = TS-2000).
func (c *Client) GetID() (string, error) {
	return c.Query("ID")
}

// SetPowerOn turns Thetis's radio engine on/off — this is the software-side
// "power" (the main Power button, console.PowerOn / chkPower), which starts
// or stops the HPSDR hardware connection and DSP audio engine. It does NOT
// touch mains power to the radio hardware itself — if the physical HL2 board
// has no mains/PoE power at all, this cannot bring it up. Wire command ZZPS
// (CATCommands.cs:5722-5744, active); the legacy Kenwood "PS" command is a
// disabled stub in this codebase and does nothing.
func (c *Client) SetPowerOn(on bool) error {
	return c.Set("ZZPS", boolDigit(on))
}

// GetPowerOn reads whether Thetis's radio engine is powered on.
func (c *Client) GetPowerOn() (bool, error) {
	reply, err := c.Query("ZZPS")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetQuickPlay starts/stops Thetis's Quick Play feature (the ckQuickPlay
// checkbox) — plays a fixed file (Music\Thetis\quickrecord\SDRQuickAudio.wav
// by convention) as I/Q injected at the head of the RX DSP chain, bypassing
// the antenna entirely; see Tools/FreeDV/README.md for how the FreeDV bench
// test-signal generator uses this for controlled, RF-free decode testing.
// If the file doesn't exist or playback otherwise fails, Thetis shows a
// local error dialog and the checkbox reverts — this call itself won't
// error in that case, since the set is fire-and-forget over CAT; confirm
// with GetQuickPlay afterward if you need to know it actually started.
// Wire command ZZQA (CATCommands.cs:7289-7311), newly wired into
// CATParser.cs's dispatch — previously unreachable dead code, found while
// investigating remote FreeDV decode testing (2026-07-30).
func (c *Client) SetQuickPlay(on bool) error {
	return c.Set("ZZQA", boolDigit(on))
}

// GetQuickPlay reads whether Quick Play is currently active.
func (c *Client) GetQuickPlay() (bool, error) {
	reply, err := c.Query("ZZQA")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetQuickRec starts/stops Thetis's Quick Rec feature (the ckQuickRec
// checkbox) — records audio to a fixed file for later Quick Play. Same
// fire-and-forget caveat as SetQuickPlay. Wire command ZZQB
// (CATCommands.cs:7312-7334), newly wired into CATParser.cs's dispatch —
// previously unreachable dead code (see SetQuickPlay).
func (c *Client) SetQuickRec(on bool) error {
	return c.Set("ZZQB", boolDigit(on))
}

// GetQuickRec reads whether Quick Rec is currently active.
func (c *Client) GetQuickRec() (bool, error) {
	reply, err := c.Query("ZZQB")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// SetFreeDVDecode enables/disables Thetis's FreeDV RX decode block (fdv.c),
// RX1/subrx0 only — matches the Setup DSP tab's current single-channel
// prototype scope (console.radio.GetDSPRX(0, 0).RXAFDVRun). Once enabled and
// synced, decoded speech replaces what RX audio you hear/capture; see
// GetFreeDVStatus for sync/SNR. Wire command ZZDV, added 2026-07-30
// alongside GetFreeDVStatus specifically to make remote FreeDV decode
// testing possible (combine with SetQuickPlay to inject a known test
// signal, then poll GetFreeDVStatus for sync).
func (c *Client) SetFreeDVDecode(on bool) error {
	return c.Set("ZZDV", boolDigit(on))
}

// SetTCIServer starts/stops Thetis's TCI server (Setup's "TCI Server"
// checkbox) via ZZTC. Not TX-capable. Useful when TCI itself is
// unreachable (e.g. right after a restart left the checkbox unchecked) -
// CAT doesn't depend on TCI being up, so this can bootstrap it back on.
func (c *Client) SetTCIServer(on bool) error {
	return c.Set("ZZTC", boolDigit(on))
}

// GetTCIServer reads whether Thetis's TCI server is currently listening.
func (c *Client) GetTCIServer() (bool, error) {
	reply, err := c.Query("ZZTC")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// GetFreeDVDecode reads whether FreeDV RX decode is currently enabled.
func (c *Client) GetFreeDVDecode() (bool, error) {
	reply, err := c.Query("ZZDV")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// FreeDVStatus is the parsed form of the ZZDS reply.
type FreeDVStatus struct {
	Sync  bool
	SNRdB float64 // only meaningful when Sync is true; 0 otherwise
}

// GetFreeDVStatus reads FreeDV RX decode sync/SNR status. Wire command ZZDS
// (get-only; CATCommands.cs's ZZDS, mirroring the Setup DSP tab's
// freedvStatusTimer_Tick — same WDSP.GetRXAFDVSync/GetRXAFDVSnr calls),
// reply format "<sync 0|1><sign><snr*10, 3 digits>" e.g. "1+153" = synced,
// 15.3dB SNR; "0+000" = not synced.
func (c *Client) GetFreeDVStatus() (FreeDVStatus, error) {
	reply, err := c.Query("ZZDS")
	if err != nil {
		return FreeDVStatus{}, err
	}
	if len(reply) != 5 {
		return FreeDVStatus{}, fmt.Errorf("cat: ZZDS reply %q: want 5 chars, got %d", reply, len(reply))
	}
	sync, err := digitBool(reply[0:1])
	if err != nil {
		return FreeDVStatus{}, fmt.Errorf("cat: ZZDS reply %q: %w", reply, err)
	}
	tenths, err := strconv.Atoi(reply[1:5])
	if err != nil {
		return FreeDVStatus{}, fmt.Errorf("cat: ZZDS reply %q: parse SNR: %w", reply, err)
	}
	return FreeDVStatus{Sync: sync, SNRdB: float64(tenths) / 10.0}, nil
}

// SetRadaeDecode enables/disables Thetis's RADE V1 RX decode block
// (ChannelMaster/radae.c), RX1 only — matches the current RX-only prototype
// scope (console.radio.GetDSPRX(0, 0).RXRadaeEnabled). Unlike FreeDV's fdv.c
// (SetFreeDVDecode above), this is inert by default: the underlying
// model/pipeline isn't wired for a full decode yet, so GetRadaeStatus is
// expected to report "no sync" even when this is on — that's the very thing
// the off-air sanity check exists to (dis)confirm. Wire command ZZDW, added
// 2026-08-10 alongside GetRadaeStatus specifically to make that check
// possible without a debugger (combine with SetQuickPlay to inject a
// captured off-air signal, then poll GetRadaeStatus for sync).
func (c *Client) SetRadaeDecode(on bool) error {
	return c.Set("ZZDW", boolDigit(on))
}

// GetRadaeDecode reads whether RADE V1 RX decode is currently enabled.
func (c *Client) GetRadaeDecode() (bool, error) {
	reply, err := c.Query("ZZDW")
	if err != nil {
		return false, err
	}
	return digitBool(reply)
}

// RadaeStatus is the parsed form of the ZZDZ reply.
type RadaeStatus struct {
	Sync  bool
	SNRdB int // only meaningful when Sync is true; 0 otherwise
}

// GetRadaeStatus reads RADE RX decode sync/SNR status. Wire command ZZDZ
// (get-only; CATCommands.cs's ZZDZ, calling WDSP.GetRadaeSync/GetRadaeSnrDb
// directly — ChannelMaster's plain rx index, not a wdsp channel), reply
// format "<sync 0|1><sign><snr dB, 3 digits>" e.g. "1+012" = synced, 12dB
// SNR; "0+000" = not synced. Unlike GetFreeDVStatus's ZZDS, the SNR digits
// are already whole dB (GetRadaeSnrDb has no *10 scaling), so no /10 here.
func (c *Client) GetRadaeStatus() (RadaeStatus, error) {
	reply, err := c.Query("ZZDZ")
	if err != nil {
		return RadaeStatus{}, err
	}
	if len(reply) != 5 {
		return RadaeStatus{}, fmt.Errorf("cat: ZZDZ reply %q: want 5 chars, got %d", reply, len(reply))
	}
	sync, err := digitBool(reply[0:1])
	if err != nil {
		return RadaeStatus{}, fmt.Errorf("cat: ZZDZ reply %q: %w", reply, err)
	}
	snr, err := strconv.Atoi(reply[1:5])
	if err != nil {
		return RadaeStatus{}, fmt.Errorf("cat: ZZDZ reply %q: parse SNR: %w", reply, err)
	}
	return RadaeStatus{Sync: sync, SNRdB: snr}, nil
}

// IFStatus is the parsed form of the Kenwood "IF" composite status reply
// (CATCommands.cs:321-402, 35 ASCII bytes).
type IFStatus struct {
	FreqHz   uint64
	RITXITHz int
	RIT      bool
	XIT      bool
	TXActive bool
	Mode     string
	Split    bool
}

// GetIF reads and parses the composite "IF" status string.
func (c *Client) GetIF() (IFStatus, error) {
	reply, err := c.Query("IF")
	if err != nil {
		return IFStatus{}, err
	}
	return parseIF(reply)
}

// parseIF decodes the 35-byte IF payload per the field layout built in
// CATCommands.cs IF() (lines 378-401):
//
//	[0:11]  freq (11 digits)      [11:15] step (4)   [15:21] RIT/XIT incr (6, signed)
//	[21:22] RIT flag              [22:23] XIT flag   [23:26] dummy (3)
//	[26:27] TX flag               [27:28] mode code  [28:30] dummy (2)
//	[30:31] split flag            [31:35] dummy (4)
func parseIF(s string) (IFStatus, error) {
	if len(s) < 31 {
		return IFStatus{}, fmt.Errorf("cat: IF reply too short: %q", s)
	}
	var st IFStatus

	freq, err := strconv.ParseUint(strings.TrimSpace(s[0:11]), 10, 64)
	if err != nil {
		return IFStatus{}, fmt.Errorf("cat: IF freq field %q: %w", s[0:11], err)
	}
	st.FreqHz = freq

	if incr, err := strconv.Atoi(s[15:21]); err == nil {
		st.RITXITHz = incr
	}
	st.RIT = s[21:22] == "1"
	st.XIT = s[22:23] == "1"
	st.TXActive = s[26:27] == "1"
	if mode, ok := codeToMode[s[27:28]]; ok {
		st.Mode = mode
	}
	st.Split = s[30:31] == "1"

	return st, nil
}

// SetPTT keys (TX) or unkeys (RX) the transmitter via the bare "TX;"/"RX;"
// CAT commands (CATCommands.cs:839-844, 987-992). This is a raw wire action
// with no safety gating of its own — callers (the thetisctl CLI) must apply
// the TX confirmation gate (internal/safety) before calling this.
func (c *Client) SetPTT(on bool) error {
	if on {
		return c.Send("TX")
	}
	return c.Send("RX")
}
