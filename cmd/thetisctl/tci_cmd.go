package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"thetisctl/internal/safety"
	"thetisctl/internal/tci"
)

func runTCI(rawArgs []string) error {
	a := parseArgs(rawArgs)
	if len(a.pos) == 0 {
		return fmt.Errorf("tci: missing command; run 'thetisctl help'")
	}
	host := a.flag("host", "")
	if host == "" {
		return fmt.Errorf("tci: --host is required (Thetis is a separate, remote instance — never assume localhost)")
	}
	port := a.flag("port", "50001")
	timeout := parseDuration(a.flag("timeout", "5s"), 5*time.Second)

	addr := net.JoinHostPort(host, port)
	conn, err := tci.Dial(addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := tci.NewClient(conn)

	cmd, args := a.pos[0], a.pos[1:]
	switch cmd {
	case "vfo":
		return tciVFO(client, args)
	case "modulation":
		return tciSet2("modulation", args, client.SetModulation)
	case "split":
		return tciToggle("split", args, client.SetSplitEnable)
	case "rit":
		return tciToggle("rit", args, client.SetRITEnable)
	case "xit":
		return tciToggle("xit", args, client.SetXITEnable)
	case "rit-offset":
		return tciSetIntInt("rit-offset", args, client.SetRITOffsetHz)
	case "xit-offset":
		return tciSetIntInt("xit-offset", args, client.SetXITOffsetHz)
	case "filter":
		return tciFilter(client, args)
	case "atten":
		return tciSetIntInt("atten", args, client.SetStepAttenuatorDB)
	case "preamp":
		return tciSetIntInt("preamp", args, client.SetPreampAttenuatorDB)
	case "agc":
		return tciSet2("agc", args, client.SetAGCMode)
	case "agc-gain":
		return tciSetIntInt("agc-gain", args, client.SetAGCGain)
	case "drive":
		return tciSetIntInt("drive", args, client.SetDrive)
	case "power":
		return tciPower(client, args, a)
	case "tune":
		return tciTune(client, args, a)
	case "ptt":
		return tciPTT(client, args, a)
	case "rx-audio":
		return tciRxAudio(client, args, a)
	case "freedv-scan":
		return tciFreeDVScan(client, a)
	case "tx-audio":
		return tciTxAudio(client, args, a)
	case "cw":
		return tciCW(client, args, a)
	case "dds":
		return tciDDS(client, args)
	case "if-offset":
		return tciIFOffset(client, args)
	case "rx-enable":
		return tciToggle("rx-enable", args, client.SetRXEnable)
	case "rx-channel":
		return tciRXChannel(client, args)
	case "iq":
		return tciIQ(client, args, a)
	case "iq-samplerate":
		return tciSetInt("iq-samplerate", args, client.SetIQSampleRateHz)
	case "audio-samplerate":
		return tciSetInt("audio-samplerate", args, client.SetAudioSampleRateHz)
	case "spot":
		return tciSpot(client, args)
	case "volume":
		return tciSetFloat("volume", args, client.SetVolumeDB)
	case "mute":
		return tciToggleGlobal("mute", args, client.SetMute)
	case "rx-mute":
		return tciToggle("rx-mute", args, client.SetRXMute)
	case "sql":
		return tciToggle("sql", args, client.SetSquelchEnable)
	case "sql-level":
		return tciSetIntInt("sql-level", args, client.SetSquelchLevelDB)
	case "rx-volume":
		return tciRXFloat3("rx-volume", args, client.SetRXVolumeDB)
	case "rx-balance":
		return tciRXFloat3("rx-balance", args, client.SetRXBalance)
	case "tune-drive":
		return tciSetIntInt("tune-drive", args, client.SetTuneDrive)
	case "rx-sensors":
		return tciSensorsEnable("rx-sensors", args, a, client.SetRXSensorsEnable)
	case "tx-sensors":
		return tciSensorsEnable("tx-sensors", args, a, client.SetTXSensorsEnable)
	case "nb":
		return tciToggle("nb", args, client.SetNoiseBlankerEnable)
	case "bin":
		return tciToggle("bin", args, client.SetBinauralEnable)
	case "nr":
		return tciToggle("nr", args, client.SetNoiseReductionEnable)
	case "anf":
		return tciToggle("anf", args, client.SetAutoNotchEnable)
	case "apf":
		return tciToggle("apf", args, client.SetAudioPeakFilterEnable)
	case "nf":
		return tciToggle("nf", args, client.SetNotchFilterEnable)
	case "focus":
		return tciFocus(client, args)
	case "query":
		return tciQuery(client, args)
	default:
		return fmt.Errorf("tci: unknown command %q", cmd)
	}
}

func parseRx(s string) (int, error) {
	rx, err := strconv.Atoi(s)
	if err != nil || (rx != 0 && rx != 1) {
		return 0, fmt.Errorf("rx must be 0 (RX1) or 1 (RX2), got %q", s)
	}
	return rx, nil
}

func tciVFO(client *tci.Client, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("vfo: usage: vfo <rx> <chan 0|1> <hz>")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("vfo: %w", err)
	}
	ch, err := strconv.Atoi(args[1])
	if err != nil || (ch != 0 && ch != 1) {
		return fmt.Errorf("vfo: chan must be 0 (VFO A) or 1 (VFO B), got %q", args[1])
	}
	hz, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("vfo: invalid Hz value %q: %w", args[2], err)
	}
	if err := client.SetVFOFreqHz(rx, ch, hz); err != nil {
		return err
	}
	fmt.Printf("vfo %d chan %d set to %d Hz\n", rx, ch, hz)
	return nil
}

func tciSet2(name string, args []string, set func(int, string) error) error {
	if len(args) != 2 {
		return fmt.Errorf("%s: usage: %s <rx> <value>", name, name)
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := set(rx, args[1]); err != nil {
		return err
	}
	fmt.Printf("%s %d set to %s\n", name, rx, args[1])
	return nil
}

func tciSetIntInt(name string, args []string, set func(int, int) error) error {
	if len(args) != 2 {
		return fmt.Errorf("%s: usage: %s <rx> <value>", name, name)
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	v, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("%s: invalid value %q: %w", name, args[1], err)
	}
	if err := set(rx, v); err != nil {
		return err
	}
	fmt.Printf("%s %d set to %d\n", name, rx, v)
	return nil
}

func tciToggle(name string, args []string, set func(int, bool) error) error {
	if len(args) != 2 {
		return fmt.Errorf("%s: usage: %s <rx> on|off", name, name)
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	var on bool
	switch args[1] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("%s: unknown value %q (want on|off)", name, args[1])
	}
	if err := set(rx, on); err != nil {
		return err
	}
	fmt.Printf("%s %d set to %v\n", name, rx, on)
	return nil
}

func tciFilter(client *tci.Client, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("filter: usage: filter <rx> <lowHz> <highHz>")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("filter: %w", err)
	}
	low, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("filter: invalid lowHz %q: %w", args[1], err)
	}
	high, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("filter: invalid highHz %q: %w", args[2], err)
	}
	if err := client.SetFilterBand(rx, low, high); err != nil {
		return err
	}
	fmt.Printf("filter %d set to [%d, %d] Hz\n", rx, low, high)
	return nil
}

func tciSetInt(name string, args []string, set func(int) error) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s <value>", name, name)
	}
	v, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%s: invalid value %q: %w", name, args[0], err)
	}
	if err := set(v); err != nil {
		return err
	}
	fmt.Printf("%s set to %d\n", name, v)
	return nil
}

func tciSetFloat(name string, args []string, set func(float64) error) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s <value>", name, name)
	}
	v, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return fmt.Errorf("%s: invalid value %q: %w", name, args[0], err)
	}
	if err := set(v); err != nil {
		return err
	}
	fmt.Printf("%s set to %g\n", name, v)
	return nil
}

func tciToggleGlobal(name string, args []string, set func(bool) error) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s on|off", name, name)
	}
	var on bool
	switch args[0] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("%s: unknown value %q (want on|off)", name, args[0])
	}
	if err := set(on); err != nil {
		return err
	}
	fmt.Printf("%s set to %v\n", name, on)
	return nil
}

// tciRXFloat3 covers <name> <rx> <chan> <float> commands (rx-volume,
// rx-balance).
func tciRXFloat3(name string, args []string, set func(int, int, float64) error) error {
	if len(args) != 3 {
		return fmt.Errorf("%s: usage: %s <rx> <chan> <value>", name, name)
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	chanNum, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("%s: invalid chan %q: %w", name, args[1], err)
	}
	v, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return fmt.Errorf("%s: invalid value %q: %w", name, args[2], err)
	}
	if err := set(rx, chanNum, v); err != nil {
		return err
	}
	fmt.Printf("%s %d chan %d set to %g\n", name, rx, chanNum, v)
	return nil
}

func tciDDS(client *tci.Client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("dds: usage: dds <rx> <hz>")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("dds: %w", err)
	}
	hz, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("dds: invalid Hz value %q: %w", args[1], err)
	}
	if err := client.SetDDSFreqHz(rx, hz); err != nil {
		return err
	}
	fmt.Printf("dds %d (panorama center) set to %d Hz — VFO moved along with it to preserve IF offset\n", rx, hz)
	return nil
}

func tciIFOffset(client *tci.Client, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("if-offset: usage: if-offset <rx> <chan 0|1> <offsetHz>")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("if-offset: %w", err)
	}
	ch, err := strconv.Atoi(args[1])
	if err != nil || (ch != 0 && ch != 1) {
		return fmt.Errorf("if-offset: chan must be 0 (VFO A) or 1 (VFO B), got %q", args[1])
	}
	hz, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("if-offset: invalid offsetHz %q: %w", args[2], err)
	}
	if err := client.SetIFOffsetHz(rx, ch, hz); err != nil {
		return err
	}
	fmt.Printf("if-offset %d chan %d set to %d Hz from DDS center\n", rx, ch, hz)
	return nil
}

func tciRXChannel(client *tci.Client, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("rx-channel: usage: rx-channel <rx> <chan 0|1> on|off")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("rx-channel: %w", err)
	}
	ch, err := strconv.Atoi(args[1])
	if err != nil || (ch != 0 && ch != 1) {
		return fmt.Errorf("rx-channel: chan must be 0 or 1, got %q", args[1])
	}
	var on bool
	switch args[2] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("rx-channel: unknown value %q (want on|off)", args[2])
	}
	if err := client.SetRXChannelEnable(rx, ch, on); err != nil {
		return err
	}
	fmt.Printf("rx-channel %d chan %d set to %v\n", rx, ch, on)
	return nil
}

func tciFocus(client *tci.Client, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("focus: takes no arguments")
	}
	if err := client.SetInFocus(); err != nil {
		return err
	}
	fmt.Println("focus: brought Thetis's main window to the foreground")
	return nil
}

func tciSensorsEnable(name string, args []string, a parsedArgs, set func(bool, int) error) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s on|off [--interval <ms>]", name, name)
	}
	var on bool
	switch args[0] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("%s: unknown value %q (want on|off)", name, args[0])
	}
	interval := 0
	if s := a.flag("interval", ""); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("%s: invalid --interval %q: %w", name, s, err)
		}
		interval = v
	}
	if err := set(on, interval); err != nil {
		return err
	}
	fmt.Printf("%s set to %v\n", name, on)
	return nil
}

// tciSpot implements the spot command group: send/delete/clear.
func tciSpot(client *tci.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("spot: usage: spot send <callsign> <mode> <hz> <argb> [extra] | spot delete <callsign> | spot clear")
	}
	switch args[0] {
	case "send":
		rest := args[1:]
		if len(rest) < 4 {
			return fmt.Errorf("spot send: usage: spot send <callsign> <mode> <hz> <argb> [extra text]")
		}
		hz, err := strconv.ParseInt(rest[2], 10, 64)
		if err != nil {
			return fmt.Errorf("spot send: invalid hz %q: %w", rest[2], err)
		}
		argb, err := strconv.ParseUint(rest[3], 10, 32)
		if err != nil {
			return fmt.Errorf("spot send: invalid argb %q: %w", rest[3], err)
		}
		extra := ""
		if len(rest) > 4 {
			extra = strings.Join(rest[4:], " ")
		}
		if err := client.SendSpot(rest[0], rest[1], hz, uint32(argb), extra); err != nil {
			return err
		}
		fmt.Printf("spot sent: %s %s %d Hz\n", rest[0], rest[1], hz)
		return nil
	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("spot delete: usage: spot delete <callsign>")
		}
		if err := client.DeleteSpot(args[1]); err != nil {
			return err
		}
		fmt.Printf("spot %s deleted\n", args[1])
		return nil
	case "clear":
		if len(args) != 1 {
			return fmt.Errorf("spot clear: takes no arguments")
		}
		if err := client.ClearSpots(); err != nil {
			return err
		}
		fmt.Println("spot: all cleared")
		return nil
	default:
		return fmt.Errorf("spot: unknown subcommand %q (want send|delete|clear)", args[0])
	}
}

// tciIQ implements "iq capture|stream <rx>", the IQ analogue of rx-audio.
// Reuses the same binary-frame plumbing as RX audio (RecvAudioFrame,
// DecodeSamples) filtered to StreamType == StreamIQ; captured/streamed data
// is 2-channel (I, Q interleaved) same as a stereo WAV, so WriteWAV's
// existing channel handling applies unchanged.
func tciIQ(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) < 1 {
		return fmt.Errorf("iq: usage: iq capture|stream <rx> [--duration 10s] [--out file.wav]")
	}
	mode, rest := args[0], args[1:]
	if mode != "capture" && mode != "stream" {
		return fmt.Errorf("iq: unknown mode %q (want capture|stream)", mode)
	}
	if len(rest) != 1 {
		return fmt.Errorf("iq %s: usage: iq %s <rx> [--duration 10s] [--out file.wav]", mode, mode)
	}
	rx, err := parseRx(rest[0])
	if err != nil {
		return fmt.Errorf("iq: %w", err)
	}
	duration := parseDuration(a.flag("duration", "10s"), 10*time.Second)

	if mode == "capture" && a.flag("out", "") == "" {
		return fmt.Errorf("iq capture: --out <file.wav> is required")
	}

	// No SetAudioSampleType call here, unlike rx-audio: IQ frames are
	// hard-coded FLOAT32 by Thetis (PublishIQSamples, TCIServer.cs:5832-5840)
	// regardless of audio_stream_sample_type, which only governs RX/TX
	// *audio* streams. DecodeSamples below is still driven by each frame's
	// own h.SampleType field, so this decodes correctly either way — the
	// difference is only that thetisctl doesn't pretend a --sample-type flag
	// would change anything for IQ.
	if mode == "stream" {
		if err := client.StartIQ(rx); err != nil {
			return err
		}
		defer client.StopIQ(rx)

		deadline := time.Now().Add(duration)
		for time.Now().Before(deadline) {
			h, data, err := client.RecvAudioFrame()
			if err != nil {
				if isTimeoutErr(err) {
					continue
				}
				return err
			}
			if h.StreamType != tci.StreamIQ || h.ReceiverID != rx {
				continue
			}
			samples := tci.DecodeSamples(data, h.SampleType)
			if err := writeFloat32LE(os.Stdout, samples); err != nil {
				return err
			}
		}
		return nil
	}

	out := a.flag("out", "")
	if err := client.StartIQ(rx); err != nil {
		return err
	}
	defer client.StopIQ(rx)

	format := tci.WAVFormat{SampleRate: 48000, Channels: 2, BitsPerSample: 32, Float: true}
	var buffered []float32
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		h, data, err := client.RecvAudioFrame()
		if err != nil {
			if isTimeoutErr(err) {
				continue
			}
			return err
		}
		if h.StreamType != tci.StreamIQ || h.ReceiverID != rx {
			continue
		}
		if h.SampleRate > 0 {
			format.SampleRate = h.SampleRate
		}
		if h.Channels > 0 {
			format.Channels = h.Channels
		}
		buffered = append(buffered, tci.DecodeSamples(data, h.SampleType)...)
	}
	if err := tci.WriteWAV(out, format, buffered); err != nil {
		return err
	}
	fmt.Printf("captured %.2fs (%d samples, %d Hz, %d ch) to %s\n",
		float64(len(buffered))/float64(format.Channels)/float64(format.SampleRate), len(buffered), format.SampleRate, format.Channels, out)
	return nil
}

// confirmTCIUnkeyed sends an unkey-type command via send and verifies, via a
// "<queryCmd>:<rx>;" query, that it actually took effect, retrying send if
// not yet confirmed. This exists because sending a command and closing the
// connection immediately afterward — every TX-capable command's original
// behavior — was proven, by direct testing against a real radio, to
// sometimes silently drop the command: Thetis does not always finish
// processing it before the socket closes. The identical unkey command sent
// over a connection kept open a couple of seconds afterward worked
// reliably every time; immediate-close dropped it more than once in a row,
// leaving the radio keyed with no time bound until a human noticed and
// intervened manually. Never trust a fire-and-forget send for anything that
// unkeys the transmitter — always confirm.
func confirmTCIUnkeyed(client *tci.Client, queryCmd string, rx int, send func() error, timeout time.Duration) error {
	rxStr := strconv.Itoa(rx)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := send(); err != nil {
			return err
		}
		checkDeadline := time.Now().Add(700 * time.Millisecond)
		for time.Now().Before(checkDeadline) {
			if err := client.SendCmd(queryCmd, rxStr); err != nil {
				return err
			}
			cmd, args, err := client.RecvCmd()
			if err != nil {
				if isTimeoutErr(err) {
					continue
				}
				return err
			}
			if cmd == queryCmd && len(args) >= 2 && args[0] == rxStr {
				if args[1] == "false" {
					return nil
				}
				break // got a reply, but still keyed — resend and retry
			}
		}
	}
	return fmt.Errorf("could not confirm rx %d unkeyed (query %q) within %s — radio may still be keyed, check manually", rx, queryCmd, timeout)
}

// tciTune and tciPTT are TCI's TX-capable commands: both gate real keying
// behind the safety confirmation phrase and auto-unkey after --hold.

// tuneMaxHold and tuneUnkeyConfirmBudget bound TUNE's total on-time at 5s no
// matter what --hold is requested: TUNE is a bare, unmodulated carrier —
// the highest-nuisance TX-capable command to leave running — so it gets a
// tighter, non-configurable ceiling than the others (user-stated
// requirement, not derived from the general reliability fix above).
const (
	tuneMaxHold             = 3 * time.Second
	tuneUnkeyConfirmBudget  = 2 * time.Second
	pttUnkeyConfirmBudget   = 5 * time.Second
	audioUnkeyConfirmBudget = 5 * time.Second
)

func tciTune(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 2 {
		return fmt.Errorf("tune: usage: tune <rx> on|off --confirm-tx=<phrase> [--hold 3s, capped at %s]", tuneMaxHold)
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("tune: %w", err)
	}
	unkey := func() error {
		return confirmTCIUnkeyed(client, "tune", rx, func() error { return client.SetTune(rx, false) }, tuneUnkeyConfirmBudget)
	}
	if args[1] == "off" {
		return unkey()
	}
	if args[1] != "on" {
		return fmt.Errorf("tune: unknown value %q (want on|off)", args[1])
	}
	hold := parseDuration(a.flag("hold", "3s"), 3*time.Second)
	if hold > tuneMaxHold {
		hold = tuneMaxHold
	}
	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Printf("[dry-run] would send: tune:%d,true; ... (hold %s) ... tune:%d,false;\n", rx, hold, rx)
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually key the transmitter.")
		return nil
	}
	if err := client.SetTune(rx, true); err != nil {
		return err
	}
	fmt.Printf("TUNE ON (rx %d) — auto-unkeying after %s (never more than %s total)\n", rx, hold, tuneMaxHold+tuneUnkeyConfirmBudget)
	time.Sleep(hold)
	if err := unkey(); err != nil {
		return fmt.Errorf("TUNE ON succeeded but could not confirm unkey: %w", err)
	}
	fmt.Println("TUNE OFF (confirmed)")
	return nil
}

func tciPTT(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 2 {
		return fmt.Errorf("ptt: usage: ptt <rx> on|off --confirm-tx=<phrase> [--hold 3s] [--audio]")
	}
	rx, err := parseRx(args[0])
	if err != nil {
		return fmt.Errorf("ptt: %w", err)
	}
	useAudio := a.has("audio")
	setOff := func() error { return client.SetTrx(rx, false) }
	if useAudio {
		setOff = func() error { return client.SetTrxTCIAudio(rx, false) }
	}
	unkey := func() error {
		return confirmTCIUnkeyed(client, "trx", rx, setOff, pttUnkeyConfirmBudget)
	}
	if args[1] == "off" {
		return unkey()
	}
	if args[1] != "on" {
		return fmt.Errorf("ptt: unknown value %q (want on|off)", args[1])
	}
	hold := parseDuration(a.flag("hold", "3s"), 3*time.Second)
	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Printf("[dry-run] would send: trx:%d,true%s; ... (hold %s) ... trx:%d,false;\n",
			rx, audioSuffix(useAudio), hold, rx)
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually key the transmitter.")
		return nil
	}
	setOn := client.SetTrx
	if useAudio {
		setOn = client.SetTrxTCIAudio
	}
	if err := setOn(rx, true); err != nil {
		return err
	}
	fmt.Printf("PTT ON (rx %d) — auto-unkeying after %s\n", rx, hold)
	time.Sleep(hold)
	if err := unkey(); err != nil {
		return fmt.Errorf("PTT ON succeeded but could not confirm unkey: %w", err)
	}
	fmt.Println("PTT OFF (confirmed)")
	return nil
}

func audioSuffix(useAudio bool) string {
	if useAudio {
		return ",tci"
	}
	return ""
}

// tciQuery is a raw passthrough for any TCI text command not covered by a
// typed subcommand. Waits for a reply whose command name matches what was
// sent, skipping anything else — otherwise, if Thetis's optional
// "send initial state on connect" burst is still arriving, the very first
// frame received could be an unrelated leftover from that burst instead of
// the genuine reply (this was a real bug, caught by the live test suite:
// "query vfo 0 0" right after connecting returned "protocol: [ExpertSDR3
// 2.0]", the burst's first line, instead of a "vfo:..." reply). This only
// matches by command name, not by leading arguments the way internal/tci's
// test-only queryTCI does, since query accepts arbitrary commands whose
// argument shape (which args identify the request vs. carry a value) isn't
// known here — for commands with multiple simultaneous instances in flight
// (e.g. "vfo" for several rx/chan pairs), the first same-named reply wins,
// which may not always be the exact one you asked for.
func tciQuery(client *tci.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("query: usage: query <cmd> [args...]")
	}
	wantCmd := args[0]
	if err := client.SendCmd(wantCmd, args[1:]...); err != nil {
		return err
	}
	for {
		cmd, replyArgs, err := client.RecvCmd()
		if err != nil {
			if isTimeoutErr(err) {
				return fmt.Errorf("query %s: no reply matching %q within timeout — "+
					"either %q produces no reply at all (some TCI commands are fire-and-forget), "+
					"or you queried right after connecting while Thetis's initial-state burst was "+
					"still arriving; retry, or pass a longer --timeout", wantCmd, wantCmd, wantCmd)
			}
			return err
		}
		if cmd != wantCmd {
			continue
		}
		fmt.Printf("%s: %v\n", cmd, replyArgs)
		return nil
	}
}

// tciPower turns Thetis's radio engine on/off (client.SetPower — the same
// action as the main Power button, not mains power to the hardware). Unlike
// most set commands, SetPower has no synchronous reply: Thetis broadcasts an
// unsolicited "start;"/"stop;" frame to every connected TCI client once the
// change actually takes effect, so this waits for that echo rather than
// assuming the send alone means it worked. Starting the hardware connection
// can take a few seconds — pass a longer --timeout if this times out.
func tciPower(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 1 {
		return fmt.Errorf("power: usage: power on|off")
	}
	var on bool
	switch args[0] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("power: unknown value %q (want on|off)", args[0])
	}

	if err := client.SetPower(on); err != nil {
		return err
	}
	want := "stop"
	if on {
		want = "start"
	}
	for {
		cmd, _, err := client.RecvCmd()
		if err != nil {
			if isTimeoutErr(err) {
				return fmt.Errorf("power: sent %q but timed out waiting for confirmation — it may still be taking effect; check with a longer --timeout or 'thetisctl cat power get'", want)
			}
			return err
		}
		if cmd == want {
			fmt.Printf("power: %v (confirmed)\n", on)
			return nil
		}
	}
}

func tciRxAudio(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) < 1 {
		return fmt.Errorf("rx-audio: usage: rx-audio capture|stream <rx> [--duration 10s] [--out file.wav] [--sample-type float32]")
	}
	mode, rest := args[0], args[1:]
	if mode != "capture" && mode != "stream" {
		return fmt.Errorf("rx-audio: unknown mode %q (want capture|stream)", mode)
	}
	if len(rest) != 1 {
		return fmt.Errorf("rx-audio %s: usage: rx-audio %s <rx> [--duration 10s] [--out file.wav]", mode, mode)
	}
	rx, err := parseRx(rest[0])
	if err != nil {
		return fmt.Errorf("rx-audio: %w", err)
	}
	duration := parseDuration(a.flag("duration", "10s"), 10*time.Second)
	sampleType := parseSampleType(a.flag("sample-type", "float32"))

	if mode == "capture" && a.flag("out", "") == "" {
		return fmt.Errorf("rx-audio capture: --out <file.wav> is required")
	}

	if mode == "stream" {
		if err := client.SetAudioSampleType(sampleType); err != nil {
			return err
		}
		if err := client.StartAudio(rx); err != nil {
			return err
		}
		defer client.StopAudio(rx)

		deadline := time.Now().Add(duration)
		for time.Now().Before(deadline) {
			h, data, err := client.RecvAudioFrame()
			if err != nil {
				if isTimeoutErr(err) {
					continue
				}
				return err
			}
			if h.StreamType != tci.StreamRXAudio || h.ReceiverID != rx {
				continue
			}
			samples := tci.DecodeSamples(data, h.SampleType)
			if err := writeFloat32LE(os.Stdout, samples); err != nil {
				return err
			}
		}
		return nil
	}

	out := a.flag("out", "")
	format, samples, err := captureRXAudio(client, rx, sampleType, duration)
	if err != nil {
		return err
	}
	if err := tci.WriteWAV(out, format, samples); err != nil {
		return err
	}
	fmt.Printf("captured %.2fs (%d samples, %d Hz, %d ch) to %s\n",
		float64(len(samples))/float64(format.Channels)/float64(format.SampleRate), len(samples), format.SampleRate, format.Channels, out)
	return nil
}

// captureRXAudio subscribes to RX audio for rx and buffers duration's worth
// of decoded samples, returning them alongside the actual format Thetis
// reported (sample rate/channels can differ from the 48kHz/mono fallback
// used if no frame ever arrives). Shared by "rx-audio capture" and
// "freedv-scan", which both need buffered (not streamed) audio.
func captureRXAudio(client *tci.Client, rx int, sampleType tci.SampleType, duration time.Duration) (tci.WAVFormat, []float32, error) {
	if err := client.SetAudioSampleType(sampleType); err != nil {
		return tci.WAVFormat{}, nil, err
	}
	if err := client.StartAudio(rx); err != nil {
		return tci.WAVFormat{}, nil, err
	}
	defer client.StopAudio(rx)

	format := tci.WAVFormat{SampleRate: 48000, Channels: 1, BitsPerSample: 32, Float: true}
	var buffered []float32

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		h, data, err := client.RecvAudioFrame()
		if err != nil {
			if isTimeoutErr(err) {
				continue
			}
			return tci.WAVFormat{}, nil, err
		}
		if h.StreamType != tci.StreamRXAudio || h.ReceiverID != rx {
			continue
		}
		if h.SampleRate > 0 {
			format.SampleRate = h.SampleRate
		}
		if h.Channels > 0 {
			format.Channels = h.Channels
		}
		buffered = append(buffered, tci.DecodeSamples(data, h.SampleType)...)
	}
	return format, buffered, nil
}

// freqEntry is one row of freeDVCallingFrequencies.
type freqEntry struct {
	Band string
	Hz   uint64
	Mode string // "usb" or "lsb", as accepted by SetModulation
}

// freeDVCallingFrequencies are the standard FreeDV digital-voice calling
// frequencies by band (user-provided reference, saved to project memory
// 2026-07-30). QO-100 is a geostationary-satellite frequency requiring an
// external transverter — the HL2's own tunable range is 0-38.4 MHz (TCI's
// vfo_limits, confirmed live) — so it's listed for completeness but always
// skipped by freedv-scan, not attempted.
var freeDVCallingFrequencies = []freqEntry{
	{"160m", 1870000, "lsb"},
	{"80m", 3625000, "lsb"},
	{"80m", 3643000, "lsb"},
	{"80m", 3693000, "lsb"},
	{"80m", 3697000, "lsb"},
	{"80m", 3803000, "lsb"},
	{"60m", 5403500, "usb"},
	{"60m", 5368500, "usb"},
	{"40m", 7177000, "lsb"},
	{"40m", 7197000, "lsb"},
	{"20m", 14236000, "usb"},
	{"20m", 14240000, "usb"},
	{"17m", 18118000, "usb"},
	{"15m", 21313000, "usb"},
	{"12m", 24933000, "usb"},
	{"10m", 28330000, "usb"},
	{"10m", 28720000, "usb"},
	{"QO-100", 10489640000, "usb"}, // satellite — always skipped, see comment above
}

// tciFreeDVScan tunes RX1 (rx 0) to each of freeDVCallingFrequencies in
// turn, dwells long enough to capture audio, and records a WAV file per
// frequency plus a peak/RMS signal-presence summary — receive-only, never
// keys the transmitter. This does NOT identify FreeDV specifically: telling
// a real digital-voice signal apart from a mistuned SSB voice transmission
// or plain band noise from spectral shape alone proved unreliable in
// practice (see SKILL.md's gotchas) — the summary is only a prioritization
// aid for which captures are worth listening to yourself. Restores the
// radio's original frequency and mode when done.
func tciFreeDVScan(client *tci.Client, a parsedArgs) error {
	dwell := parseDuration(a.flag("dwell", "6s"), 6*time.Second)
	outDir := a.flag("out-dir", "")
	if outDir == "" {
		var err error
		outDir, err = os.MkdirTemp("", "freedv-scan-")
		if err != nil {
			return fmt.Errorf("freedv-scan: create output dir: %w", err)
		}
	} else if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("freedv-scan: create output dir: %w", err)
	}

	origArgs := queryTCICmd(client, "vfo", []string{"0", "0"})
	origFreq, haveOrigFreq := int64(0), false
	if len(origArgs) >= 3 {
		if v, err := strconv.ParseInt(origArgs[2], 10, 64); err == nil {
			origFreq, haveOrigFreq = v, true
		}
	}
	modArgs := queryTCICmd(client, "modulation", []string{"0"})
	origMode, haveOrigMode := "", false
	if len(modArgs) >= 2 {
		origMode, haveOrigMode = modArgs[1], true
	}
	if haveOrigFreq && haveOrigMode {
		defer func() {
			client.SetModulation(0, origMode)
			client.SetVFOFreqHz(0, 0, origFreq)
		}()
	}

	type result struct {
		entry freqEntry
		file  string
		peak  float32
		rms   float64
	}
	var results []result

	fmt.Printf("freedv-scan: %d frequencies, %s dwell each, saving to %s\n",
		len(freeDVCallingFrequencies)-1, dwell, outDir)

	for _, e := range freeDVCallingFrequencies {
		if e.Band == "QO-100" {
			fmt.Printf("[skip] QO-100 %d Hz — satellite frequency, outside HL2 direct-tune range (0-38.4MHz)\n", e.Hz)
			continue
		}
		if err := client.SetModulation(0, e.Mode); err != nil {
			return fmt.Errorf("freedv-scan: set modulation for %s: %w", e.Band, err)
		}
		if err := client.SetVFOFreqHz(0, 0, int64(e.Hz)); err != nil {
			return fmt.Errorf("freedv-scan: set frequency for %s: %w", e.Band, err)
		}
		time.Sleep(400 * time.Millisecond) // let AGC/DSP settle after retune

		format, samples, err := captureRXAudio(client, 0, tci.SampleFloat32, dwell)
		if err != nil {
			return fmt.Errorf("freedv-scan: capture at %d Hz: %w", e.Hz, err)
		}

		file := filepath.Join(outDir, fmt.Sprintf("%s_%dHz_%s.wav", e.Band, e.Hz, strings.ToUpper(e.Mode)))
		if err := tci.WriteWAV(file, format, samples); err != nil {
			return fmt.Errorf("freedv-scan: write %s: %w", file, err)
		}

		var peak float32
		var sumSq float64
		for _, s := range samples {
			if abs := float32(math.Abs(float64(s))); abs > peak {
				peak = abs
			}
			sumSq += float64(s) * float64(s)
		}
		rms := 0.0
		if len(samples) > 0 {
			rms = math.Sqrt(sumSq / float64(len(samples)))
		}
		fmt.Printf("[%2d/%2d] %-6s %10.4f MHz %s: peak=%.4f rms=%.4f -> %s\n",
			len(results)+1, len(freeDVCallingFrequencies)-1, e.Band, float64(e.Hz)/1e6, strings.ToUpper(e.Mode), peak, rms, file)

		results = append(results, result{e, file, peak, rms})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].rms > results[j].rms })
	fmt.Println("\nSummary, sorted by RMS (most active first — listen to these first, this is a prioritization hint, not a FreeDV identification):")
	for _, r := range results {
		fmt.Printf("  %-6s %10.4f MHz %s  peak=%.4f rms=%.4f  %s\n",
			r.entry.Band, float64(r.entry.Hz)/1e6, strings.ToUpper(r.entry.Mode), r.peak, r.rms, r.file)
	}
	return nil
}

// queryTCICmd sends "cmd:args...;" and returns the first reply's args whose
// command name matches — best-effort, no retry/burst-draining (freedv-scan
// only uses this to capture original state for later restoration, where
// "couldn't determine original state, skip restoring" is an acceptable
// fallback, unlike a query result a user is relying on for a real decision).
func queryTCICmd(client *tci.Client, cmd string, sendArgs []string) []string {
	if err := client.SendCmd(cmd, sendArgs...); err != nil {
		return nil
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		gotCmd, args, err := client.RecvCmd()
		if err != nil {
			if isTimeoutErr(err) {
				continue
			}
			return nil
		}
		if gotCmd == cmd {
			return args
		}
	}
	return nil
}

func writeFloat32LE(w *os.File, samples []float32) error {
	buf := make([]byte, len(samples)*4)
	for i, s := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(s))
	}
	_, err := w.Write(buf)
	return err
}

func isTimeoutErr(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func parseSampleType(s string) tci.SampleType {
	switch s {
	case "int16":
		return tci.SampleInt16
	case "int24":
		return tci.SampleInt24
	case "int32":
		return tci.SampleInt32
	default:
		return tci.SampleFloat32
	}
}

// tciTxAudio is TCI's most consequential TX-capable command: it streams a
// WAV file's audio as TX_AUDIO_STREAM frames while PTT is held via the
// "tci"-audio-source form of trx, which genuinely modulates and transmits
// RF (Console/cmaster.cs TCITxThreadProc drains this straight into the DSP
// TX chain). Gated behind the safety confirmation phrase; always applies a
// hard --max-duration cap and unkeys on completion, error, or Ctrl-C.
func tciTxAudio(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 2 || args[0] != "send" {
		return fmt.Errorf("tx-audio: usage: tx-audio send <rx> --file tone.wav --confirm-tx=<phrase> [--max-duration 10s] [--sample-type int16]")
	}
	rx, err := parseRx(args[1])
	if err != nil {
		return fmt.Errorf("tx-audio: %w", err)
	}
	file := a.flag("file", "")
	if file == "" {
		return fmt.Errorf("tx-audio send: --file <wav> is required")
	}
	maxDuration := parseDuration(a.flag("max-duration", "10s"), 10*time.Second)
	sampleType := parseSampleType(a.flag("sample-type", "int16"))

	format, samples, err := tci.ReadWAV(file)
	if err != nil {
		return err
	}
	if format.Channels < 1 {
		format.Channels = 1
	}
	totalDuration := time.Duration(float64(len(samples)) / float64(format.Channels) / float64(format.SampleRate) * float64(time.Second))
	truncated := false
	if totalDuration > maxDuration {
		keepFrames := int(maxDuration.Seconds() * float64(format.SampleRate))
		keepSamples := keepFrames * format.Channels
		if keepSamples < len(samples) {
			samples = samples[:keepSamples]
			truncated = true
		}
		totalDuration = maxDuration
	}
	peak := peakAbs(samples)

	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Printf("[dry-run] would send: trx:%d,true,tci; then stream %s of TX audio from %s (%d Hz, %d ch, peak %.3f%s) as %s frames; then trx:%d,false,tci;\n",
			rx, totalDuration, file, format.SampleRate, format.Channels, peak, truncatedNote(truncated), sampleType.WireName(), rx)
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually transmit this audio.")
		return nil
	}

	if err := client.SetAudioSampleType(sampleType); err != nil {
		return err
	}

	// Always unkey on completion, error, or Ctrl-C — never leave the radio
	// keyed because this process exited unexpectedly. Confirmed, not
	// fire-and-forget — see confirmTCIUnkeyed's doc comment.
	unkeyed := false
	unkey := func() {
		if unkeyed {
			return
		}
		unkeyed = true
		if err := confirmTCIUnkeyed(client, "trx", rx, func() error { return client.SetTrxTCIAudio(rx, false) }, audioUnkeyConfirmBudget); err != nil {
			fmt.Fprintln(os.Stderr, "tx-audio: WARNING:", err)
		}
	}
	defer unkey()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		if _, ok := <-sigCh; ok {
			unkey()
			fmt.Fprintln(os.Stderr, "\ntx-audio: interrupted, unkeyed")
			os.Exit(130)
		}
	}()

	if err := client.SetTrxTCIAudio(rx, true); err != nil {
		return err
	}
	fmt.Printf("TX ON (rx %d, TCI audio) — streaming %s%s\n", rx, totalDuration, truncatedNote(truncated))

	const chunkFrames = 2048
	chunkSamples := chunkFrames * format.Channels
	for off := 0; off < len(samples); off += chunkSamples {
		end := off + chunkSamples
		if end > len(samples) {
			end = len(samples)
		}
		chunk := samples[off:end]
		h := tci.StreamHeader{
			ReceiverID: rx,
			SampleRate: format.SampleRate,
			SampleType: sampleType,
			Length:     len(chunk),
			StreamType: tci.StreamTXAudio,
			Channels:   format.Channels,
		}
		if err := client.SendAudioFrame(h, tci.EncodeSamples(chunk, sampleType)); err != nil {
			return err
		}
		frames := len(chunk) / format.Channels
		time.Sleep(time.Duration(float64(frames)/float64(format.SampleRate)*1000) * time.Millisecond)
	}

	unkey()
	fmt.Println("TX OFF")
	return nil
}

func truncatedNote(truncated bool) string {
	if truncated {
		return " [truncated to --max-duration]"
	}
	return ""
}

func peakAbs(samples []float32) float32 {
	var peak float32
	for _, s := range samples {
		if s < 0 {
			s = -s
		}
		if s > peak {
			peak = s
		}
	}
	return peak
}

// tciCW dispatches the cw command group. "send" is TX-capable (see
// tciCWSend); "speed-up"/"speed-down"/"delay" adjust the CW macro/message
// keyer's timing (not TX-capable on their own); "terminal" toggles CW
// Terminal mode (config only, see SetCWTerminalEnable's doc comment);
// "send-msg" is cw_msg's TX-capable callsign-repeat form (see
// tciCWSendMsg); "edit-callsign" edits an in-progress send-msg
// transmission's callsign and is not TX-capable on its own (see
// EditCWMessageCallsign's doc comment).
func tciCW(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) < 1 {
		return fmt.Errorf("cw: usage: cw send|send-msg|edit-callsign|speed-up|speed-down|delay|terminal ...")
	}
	switch args[0] {
	case "send":
		return tciCWSend(client, args, a)
	case "send-msg":
		return tciCWSendMsg(client, args, a)
	case "edit-callsign":
		if len(args) != 2 {
			return fmt.Errorf("cw edit-callsign: usage: cw edit-callsign <callsign>")
		}
		if err := client.EditCWMessageCallsign(args[1]); err != nil {
			return err
		}
		fmt.Printf("cw: in-progress message callsign updated to %s\n", args[1])
		return nil
	case "speed-up":
		return tciCWSpeedStep("cw speed-up", args[1:], client.IncreaseCWMacroSpeedWPM)
	case "speed-down":
		return tciCWSpeedStep("cw speed-down", args[1:], client.DecreaseCWMacroSpeedWPM)
	case "delay":
		return tciSetInt("cw delay", args[1:], client.SetCWMacroDelayMs)
	case "terminal":
		return tciToggle("cw terminal", args[1:], client.SetCWTerminalEnable)
	default:
		return fmt.Errorf("cw: unknown subcommand %q (want send|send-msg|edit-callsign|speed-up|speed-down|delay|terminal)", args[0])
	}
}

func tciCWSpeedStep(name string, args []string, step func(int) error) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s <wpm amount>", name, name)
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%s: invalid amount %q: %w", name, args[0], err)
	}
	if err := step(amount); err != nil {
		return err
	}
	fmt.Printf("%s: %d WPM\n", name, amount)
	return nil
}

// tciCWSendMsg is cw_msg's TX-capable form: like tciCWSend, but with
// separate before/callsign/after fields so Thetis's engine can repeat or
// mid-transmission-edit the callsign (see SendCWMessage's doc comment).
// Shares tciCWSend's TX safety shape (dry-run by default, confirmed unkey,
// PTT-poll completion detection) rather than cw_macros_empty, since this
// command doesn't enable CW Terminal mode either.
func tciCWSendMsg(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 5 {
		return fmt.Errorf("cw send-msg: usage: cw send-msg <rx> <prefix|_> <callsign> <suffix> --confirm-tx=<phrase> [--speed 20] [--mode cw|cwu|cwl] [--max-duration 90s]")
	}
	rx, err := parseRx(args[1])
	if err != nil {
		return fmt.Errorf("cw send-msg: %w", err)
	}
	prefix, callsign, suffix := args[2], args[3], args[4]
	if callsign == "" {
		return fmt.Errorf("cw send-msg: callsign must not be empty")
	}
	speed, err := strconv.Atoi(a.flag("speed", "20"))
	if err != nil || speed < 1 || speed > 99 {
		return fmt.Errorf("cw send-msg: --speed must be an integer 1-99")
	}
	mode := a.flag("mode", "cw")
	if mode != "cw" && mode != "cwu" && mode != "cwl" {
		return fmt.Errorf("cw send-msg: --mode must be cw, cwu, or cwl")
	}
	maxDuration := parseDuration(a.flag("max-duration", "90s"), 90*time.Second)

	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Printf("[dry-run] would send: modulation:%d,%s; cw_macros_speed:%d; cw_msg:%d,%s,%s,%s;\n", rx, mode, speed, rx, prefix, callsign, suffix)
		fmt.Printf("(Thetis keys PTT itself while sending; completion is detected by polling PTT state, hard-capped at %s)\n", maxDuration)
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually transmit this message.")
		return nil
	}

	if err := client.SetModulation(rx, mode); err != nil {
		return err
	}
	if err := client.SetCWMacroSpeedWPM(speed); err != nil {
		return err
	}

	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		if err := confirmTCIUnkeyed(client, "trx", rx, client.StopCWMacros, pttUnkeyConfirmBudget); err != nil {
			fmt.Fprintln(os.Stderr, "cw send-msg: WARNING:", err)
		}
	}
	defer stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		if _, ok := <-sigCh; ok {
			stop()
			fmt.Fprintln(os.Stderr, "\ncw send-msg: interrupted, sent cw_macros_stop")
			os.Exit(130)
		}
	}()

	if err := client.SendCWMessage(rx, prefix, callsign, suffix); err != nil {
		return err
	}
	fmt.Printf("CW ON (rx %d, %d WPM) — sending %q/%q/%q, waiting for completion (max %s)\n", rx, speed, prefix, callsign, suffix, maxDuration)

	rxStr := strconv.Itoa(rx)
	everKeyed := false
	lastPoll := time.Time{}
	deadline := time.Now().Add(maxDuration)
	for time.Now().Before(deadline) {
		if time.Since(lastPoll) > 500*time.Millisecond {
			if err := client.SendCmd("trx", rxStr); err != nil {
				return err
			}
			lastPoll = time.Now()
		}
		cmd, cmdArgs, err := client.RecvCmd()
		if err != nil {
			if isTimeoutErr(err) {
				continue
			}
			return err
		}
		if cmd != "trx" || len(cmdArgs) < 2 || cmdArgs[0] != rxStr {
			continue
		}
		if cmdArgs[1] == "true" {
			everKeyed = true
			continue
		}
		if everKeyed {
			fmt.Println("CW done (PTT released)")
			return nil
		}
	}

	stop()
	if !everKeyed {
		return fmt.Errorf("cw send-msg: timed out after %s and PTT was never observed keyed — message likely did not send (another client may own the CW engine); sent cw_macros_stop as a precaution", maxDuration)
	}
	return fmt.Errorf("cw send-msg: timed out after %s waiting for PTT to release after keying; sent cw_macros_stop", maxDuration)
}

// tciCWSend is TCI's other TX-capable command: it hands free text to Thetis's
// own CW macro/keyer engine, which manages PTT/MOX itself while it keys the
// message (TCICWController.SendMacro, TCIServer.cs:8449-8462) — genuinely
// transmitting RF. Gated behind the safety confirmation phrase; switches the
// target receiver to CW mode first, waits for the server's unsolicited
// "cw_macros_empty:<rx>;" completion notice, and always falls back to
// cw_macros_stop on timeout, error, or Ctrl-C so the radio can't be left
// keyed by a message that never finishes.
func tciCWSend(client *tci.Client, args []string, a parsedArgs) error {
	if len(args) != 3 || args[0] != "send" {
		return fmt.Errorf("cw: usage: cw send <rx> <text> --confirm-tx=<phrase> [--speed 20] [--mode cw|cwu|cwl] [--max-duration 90s]")
	}
	rx, err := parseRx(args[1])
	if err != nil {
		return fmt.Errorf("cw: %w", err)
	}
	text := args[2]
	if text == "" {
		return fmt.Errorf("cw send: text must not be empty")
	}
	speed, err := strconv.Atoi(a.flag("speed", "20"))
	if err != nil || speed < 1 || speed > 99 {
		return fmt.Errorf("cw send: --speed must be an integer 1-99")
	}
	mode := a.flag("mode", "cw")
	if mode != "cw" && mode != "cwu" && mode != "cwl" {
		return fmt.Errorf("cw send: --mode must be cw, cwu, or cwl")
	}
	maxDuration := parseDuration(a.flag("max-duration", "90s"), 90*time.Second)

	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Printf("[dry-run] would send: modulation:%d,%s; cw_macros_speed:%d; cw_macros:%d,%s;\n", rx, mode, speed, rx, text)
		fmt.Printf("(Thetis keys PTT itself while sending; completion is detected by polling PTT state, hard-capped at %s)\n", maxDuration)
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually transmit this message.")
		return nil
	}

	if err := client.SetModulation(rx, mode); err != nil {
		return err
	}
	if err := client.SetCWMacroSpeedWPM(speed); err != nil {
		return err
	}

	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		if err := confirmTCIUnkeyed(client, "trx", rx, client.StopCWMacros, pttUnkeyConfirmBudget); err != nil {
			fmt.Fprintln(os.Stderr, "cw: WARNING:", err)
		}
	}
	defer stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		if _, ok := <-sigCh; ok {
			stop()
			fmt.Fprintln(os.Stderr, "\ncw: interrupted, sent cw_macros_stop")
			os.Exit(130)
		}
	}()

	if err := client.SendCWMacro(rx, text); err != nil {
		return err
	}
	fmt.Printf("CW ON (rx %d, %d WPM) — sending %q, waiting for completion (max %s)\n", rx, speed, text, maxDuration)

	// Thetis's CW macro engine has no "message finished" event for plain
	// cw_macros sends — "cw_macros_empty" only fires in CW Terminal mode
	// (TCICWController's two OnCwMacrosEmpty call sites are both gated on
	// isTerminalEnabledLocked, TCIServer.cs:8547,8852-8853), which this
	// command doesn't enable. Instead poll the bare "trx:<rx>;" query (1 arg
	// = get, handleTrxMessage TCIServer.cs:3690-3693), which the engine's own
	// PTT/MOX state answers truthfully (sendMOX, TCIServer.cs:2159-2168):
	// wait for MOX to go true (keyed) then false (unkeyed) again.
	rxStr := strconv.Itoa(rx)
	everKeyed := false
	lastPoll := time.Time{}
	deadline := time.Now().Add(maxDuration)
	for time.Now().Before(deadline) {
		if time.Since(lastPoll) > 500*time.Millisecond {
			if err := client.SendCmd("trx", rxStr); err != nil {
				return err
			}
			lastPoll = time.Now()
		}
		cmd, cmdArgs, err := client.RecvCmd()
		if err != nil {
			if isTimeoutErr(err) {
				continue
			}
			return err
		}
		if cmd != "trx" || len(cmdArgs) < 2 || cmdArgs[0] != rxStr {
			continue
		}
		if cmdArgs[1] == "true" {
			everKeyed = true
			continue
		}
		if everKeyed {
			fmt.Println("CW done (PTT released)")
			return nil
		}
	}

	stop()
	if !everKeyed {
		return fmt.Errorf("cw send: timed out after %s and PTT was never observed keyed — message likely did not send (another client may own the CW engine); sent cw_macros_stop as a precaution", maxDuration)
	}
	return fmt.Errorf("cw send: timed out after %s waiting for PTT to release after keying; sent cw_macros_stop", maxDuration)
}
