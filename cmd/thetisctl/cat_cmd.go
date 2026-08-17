package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"thetisctl/internal/cat"
	"thetisctl/internal/safety"
)

func runCAT(rawArgs []string) error {
	a := parseArgs(rawArgs)
	if len(a.pos) == 0 {
		return fmt.Errorf("cat: missing command; run 'thetisctl help'")
	}
	host := a.flag("host", "")
	if host == "" {
		return fmt.Errorf("cat: --host is required (Thetis is a separate, remote instance — never assume localhost)")
	}
	port := a.flag("port", "13013")
	timeout := parseDuration(a.flag("timeout", "3s"), 3*time.Second)

	addr := net.JoinHostPort(host, port)
	c, err := cat.Dial(addr, timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	cmd, args := a.pos[0], a.pos[1:]
	switch cmd {
	case "freq":
		return catFreq(c, args)
	case "mode":
		return catMode(c, args)
	case "rit":
		return catToggle("rit", args, c.SetRIT, c.GetRIT)
	case "xit":
		return catToggle("xit", args, c.SetXIT, c.GetXIT)
	case "split":
		return catToggle("split", args, c.SetSplit, c.GetSplit)
	case "agc":
		return catAGC(c, args)
	case "atten":
		return catAtten(c, args)
	case "preamp":
		return catPreamp(c, args)
	case "band":
		return catBand(c, args)
	case "power":
		return catToggle("power", args, c.SetPowerOn, c.GetPowerOn)
	case "quickplay":
		return catQuickPlay(c, args, a)
	case "quickrec":
		return catToggle("quickrec", args, c.SetQuickRec, c.GetQuickRec)
	case "freedv":
		return catFreeDV(c, args)
	case "radae":
		return catRadae(c, args)
	case "radae-sanity":
		return catRadaeSanity(c, args, a)
	case "tciserver":
		return catToggle("tciserver", args, c.SetTCIServer, c.GetTCIServer)
	case "status":
		return catStatus(c)
	case "version":
		return catVersion(c)
	case "query":
		return catQuery(c, args)
	case "ptt":
		return catPTT(c, args, a)
	default:
		return fmt.Errorf("cat: unknown command %q", cmd)
	}
}

func catFreq(c *cat.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("freq: usage: freq get A|B | freq set A|B <hz>")
	}
	switch args[0] {
	case "get":
		hz, err := c.GetVFOFreqHz(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("VFO %s: %d Hz\n", strings.ToUpper(args[1]), hz)
	case "set":
		if len(args) != 3 {
			return fmt.Errorf("freq set: usage: freq set A|B <hz>")
		}
		hz, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("freq set: invalid Hz value %q: %w", args[2], err)
		}
		if err := c.SetVFOFreqHz(args[1], hz); err != nil {
			return err
		}
		got, err := c.GetVFOFreqHz(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("VFO %s set to %d Hz (confirmed: %d Hz)\n", strings.ToUpper(args[1]), hz, got)
	default:
		return fmt.Errorf("freq: unknown subcommand %q", args[0])
	}
	return nil
}

func catMode(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mode: usage: mode get | mode set <name>")
	}
	switch args[0] {
	case "get":
		mode, err := c.GetMode()
		if err != nil {
			return err
		}
		fmt.Println("Mode:", mode)
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("mode set: usage: mode set <USB|LSB|CW|CWL|FM|AM|DIGU|DIGL>")
		}
		if err := c.SetMode(args[1]); err != nil {
			return err
		}
		got, err := c.GetMode()
		if err != nil {
			return err
		}
		fmt.Println("Mode set to (confirmed):", got)
	default:
		return fmt.Errorf("mode: unknown subcommand %q", args[0])
	}
	return nil
}

// catToggle implements the common "<name> on|off|get" shape shared by
// rit/xit/split, always printing the confirmed state read back from Thetis.
func catToggle(name string, args []string, set func(bool) error, get func() (bool, error)) error {
	if len(args) != 1 {
		return fmt.Errorf("%s: usage: %s on|off|get", name, name)
	}
	switch args[0] {
	case "on":
		if err := set(true); err != nil {
			return err
		}
	case "off":
		if err := set(false); err != nil {
			return err
		}
	case "get":
		// read-only, fall through to the status print below
	default:
		return fmt.Errorf("%s: unknown value %q (want on|off|get)", name, args[0])
	}
	v, err := get()
	if err != nil {
		return err
	}
	fmt.Printf("%s: %v\n", name, v)
	return nil
}

// catFreeDV controls and reports on Thetis's FreeDV RX decode block (fdv.c).
// "status" is read-only (sync/SNR); on|off|get toggles/reads whether the
// decoder is running at all — same shape as quickplay/quickrec above.
// Combine with "quickplay on" to inject the FreeDV bench test signal and
// this to check whether it synced, entirely without touching the Thetis UI.
func catFreeDV(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("freedv: usage: freedv on|off|get | freedv status")
	}
	if args[0] == "status" {
		st, err := c.GetFreeDVStatus()
		if err != nil {
			return err
		}
		if st.Sync {
			fmt.Printf("freedv: SYNC  SNR %.1f dB\n", st.SNRdB)
		} else {
			fmt.Println("freedv: no sync")
		}
		return nil
	}
	return catToggle("freedv", args, c.SetFreeDVDecode, c.GetFreeDVDecode)
}

// catRadae controls and reports on Thetis's RADE V1 RX decode block
// (ChannelMaster/radae.c) — same shape as catFreeDV above, ZZDW/ZZDZ instead
// of ZZDV/ZZDS. "status" is read-only (sync/SNR); on|off|get toggles/reads
// whether the decoder is running at all. Combine with "quickplay on" to
// inject a captured off-air signal and this to check whether it synced —
// see radaeSanity for the scripted end-to-end version of that check.
func catRadae(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("radae: usage: radae on|off|get | radae status")
	}
	if args[0] == "status" {
		st, err := c.GetRadaeStatus()
		if err != nil {
			return err
		}
		if st.Sync {
			fmt.Printf("radae: SYNC  SNR %d dB\n", st.SNRdB)
		} else {
			fmt.Println("radae: no sync")
		}
		return nil
	}
	return catToggle("radae", args, c.SetRadaeDecode, c.GetRadaeDecode)
}

// catQuickPlay is TX-capable, discovered by live testing 2026-08-04 — do
// not revert this to catToggle. Quick Play "on" calls PlayFileViaWDSP
// (Console/clsAudioRecordPlayback.cs), a function shared with a genuine
// TX-audio-preview feature. That function contains:
//
//	if (!_console.MOX && MoxOnPlayback) _console.MOX = true;
//
// and MoxOnPlayback defaults to true in this codebase (same file,
// `public bool MoxOnPlayback { get; set; } = true;`) — controlled by
// Setup → Recording's "MOX on Playback" checkbox. Quick Play is documented
// elsewhere (and was originally documented here) as RX-only — it injects a
// wav as RX I/Q ahead of the antenna input — but that describes its
// *intent*, not a guarantee against this shared function's side effect.
// Before this fix, "on" went through catToggle with no TX gate at all:
// every call could have kept MOX regardless of --confirm-tx.
func catQuickPlay(c *cat.Client, args []string, a parsedArgs) error {
	if len(args) != 1 {
		return fmt.Errorf("quickplay: usage: quickplay on --confirm-tx=<phrase> [--hold 15s] | quickplay off | quickplay get")
	}
	switch args[0] {
	case "get":
		on, err := c.GetQuickPlay()
		if err != nil {
			return err
		}
		fmt.Printf("quickplay: %v\n", on)
		return nil
	case "off":
		// Stopping Quick Play restores whatever MOX state preceded it
		// (Console's storeRestoreSettings) — confirm that actually landed
		// rather than trusting a fire-and-forget send, same reasoning as
		// confirmCATUnkeyed elsewhere in this file.
		if err := confirmCATUnkeyed(c, func() error { return c.SetQuickPlay(false) }, 5*time.Second); err != nil {
			return fmt.Errorf("quickplay off: %w", err)
		}
		on, err := c.GetQuickPlay()
		if err != nil {
			return err
		}
		fmt.Printf("quickplay: %v\n", on)
		return nil
	case "on":
		dec := safety.Check(a.flag("confirm-tx", ""))
		hold := parseDuration(a.flag("hold", "15s"), 15*time.Second)
		if dec.DryRun {
			fmt.Println("[dry-run] would send: quickplay on")
			fmt.Println("WARNING: this may key MOX for real. PlayFileViaWDSP keys MOX whenever the")
			fmt.Println("console's \"MOX on Playback\" setting (Setup → Recording) is enabled — which")
			fmt.Println("defaults to true in this codebase. Verify that setting is off if you need this")
			fmt.Println("to be genuinely RX-only, or pass --confirm-tx if you accept it may key the radio.")
			fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to proceed.")
			return nil
		}
		if err := c.SetQuickPlay(true); err != nil {
			return err
		}
		fmt.Printf("quickplay: true — auto-stopping after %s\n", hold)
		time.Sleep(hold)
		if err := confirmCATUnkeyed(c, func() error { return c.SetQuickPlay(false) }, 5*time.Second); err != nil {
			return fmt.Errorf("quickplay on succeeded but could not confirm stop: %w", err)
		}
		fmt.Println("quickplay: false (confirmed)")
		return nil
	default:
		return fmt.Errorf("quickplay: unknown value %q (want on|off|get)", args[0])
	}
}

func catAGC(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("agc: usage: agc get | agc set <FIXED|LONG|SLOW|MEDIUM|FAST|CUSTOM>")
	}
	switch args[0] {
	case "get":
		mode, err := c.GetAGC()
		if err != nil {
			return err
		}
		fmt.Println("AGC:", mode)
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("agc set: usage: agc set <FIXED|LONG|SLOW|MEDIUM|FAST|CUSTOM>")
		}
		if err := c.SetAGC(args[1]); err != nil {
			return err
		}
		got, err := c.GetAGC()
		if err != nil {
			return err
		}
		fmt.Println("AGC set to (confirmed):", got)
	default:
		return fmt.Errorf("agc: unknown subcommand %q", args[0])
	}
	return nil
}

func catAtten(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("atten: usage: atten get | atten set <0-31>")
	}
	switch args[0] {
	case "get":
		db, err := c.GetAttenuatorDB()
		if err != nil {
			return err
		}
		fmt.Printf("Attenuator: %d dB\n", db)
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("atten set: usage: atten set <0-31>")
		}
		db, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("atten set: invalid dB value %q: %w", args[1], err)
		}
		if err := c.SetAttenuatorDB(db); err != nil {
			return err
		}
		got, err := c.GetAttenuatorDB()
		if err != nil {
			return err
		}
		fmt.Printf("Attenuator set to (confirmed): %d dB\n", got)
	default:
		return fmt.Errorf("atten: unknown subcommand %q", args[0])
	}
	return nil
}

func catPreamp(c *cat.Client, args []string) error {
	if len(args) != 2 || args[0] != "set" {
		return fmt.Errorf("preamp: usage: preamp set <0-9> (0=off 1=on 2..6=-10..-50dB 7..9=SA -10..-30dB)")
	}
	level, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("preamp set: invalid level %q: %w", args[1], err)
	}
	if err := c.SetPreamp(level); err != nil {
		return err
	}
	fmt.Println("Preamp level set to:", level)
	return nil
}

func catBand(c *cat.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("band: usage: band get | band set <name>")
	}
	switch args[0] {
	case "get":
		band, err := c.GetBand()
		if err != nil {
			return err
		}
		fmt.Println("Band:", band)
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("band set: usage: band set <160|80|60|40|30|20|17|15|12|10|6|2|GEN|WWV|V0-V13>")
		}
		if err := c.SetBand(args[1]); err != nil {
			return err
		}
		got, err := c.GetBand()
		if err != nil {
			return err
		}
		fmt.Println("Band set to (confirmed):", got)
	default:
		return fmt.Errorf("band: unknown subcommand %q", args[0])
	}
	return nil
}

// catVersion reads Thetis's own software version string via ZZZV
// ("Get Software Version String", CATStructs.xml — already implemented and
// wired into CATParser.cs's dispatch, just not previously exposed by this
// tool). As of 2026-08, the string includes the exact git short SHA the
// running build was made from (Thetis.csproj's PreBuildEvent generates
// VersionInfo.GitShortSha at build time; titlebar.cs appends it) — this is
// the way to confirm which commit is actually running on a remote
// instance, rather than assuming a build/install matches what you expect.
func catVersion(c *cat.Client) error {
	v, err := c.Query("ZZZV")
	if err != nil {
		return err
	}
	fmt.Println("version:", v)
	return nil
}

// catQuery is a raw passthrough for any CAT command not wrapped by a
// dedicated subcommand above, mirroring `tci query`. Sends "<code>;" and
// prints the reply with the code prefix stripped. Useful for one-off reads
// of commands (like ZZZV before catVersion existed) without waiting for a
// dedicated wrapper.
func catQuery(c *cat.Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("query: usage: query <CODE>  (e.g. query ZZZV)")
	}
	v, err := c.Query(args[0])
	if err != nil {
		return err
	}
	fmt.Println(args[0]+":", v)
	return nil
}

func catStatus(c *cat.Client) error {
	id, err := c.GetID()
	if err != nil {
		return err
	}
	st, err := c.GetIF()
	if err != nil {
		return err
	}
	fmt.Printf("Rig ID:   %s\n", id)
	fmt.Printf("Freq:     %d Hz\n", st.FreqHz)
	fmt.Printf("Mode:     %s\n", st.Mode)
	fmt.Printf("RIT/XIT:  RIT=%v XIT=%v offset=%+d Hz\n", st.RIT, st.XIT, st.RITXITHz)
	fmt.Printf("Split:    %v\n", st.Split)
	fmt.Printf("TX:       %v\n", st.TXActive)
	return nil
}

// confirmCATUnkeyed sends an unkey command via send and verifies, via
// GetIF's TXActive flag, that it actually took effect, retrying send if
// not yet confirmed. Same reasoning as internal/tci's confirmTCIUnkeyed —
// sending a command and closing the connection immediately afterward was
// proven, by direct testing against a real radio, to sometimes silently
// drop it. Never trust a fire-and-forget send for anything that unkeys the
// transmitter — always confirm.
func confirmCATUnkeyed(c *cat.Client, send func() error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := send(); err != nil {
			return err
		}
		checkDeadline := time.Now().Add(700 * time.Millisecond)
		for time.Now().Before(checkDeadline) {
			st, err := c.GetIF()
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !st.TXActive {
				return nil
			}
			break // got a reply, but still keyed — resend and retry
		}
	}
	return fmt.Errorf("could not confirm PTT unkeyed within %s — radio may still be keyed, check manually", timeout)
}

// catPTT is the sole TX-capable CAT command: it gates real keying behind the
// safety confirmation phrase and always auto-unkeys after --hold.
func catPTT(c *cat.Client, args []string, a parsedArgs) error {
	if len(args) != 1 {
		return fmt.Errorf("ptt: usage: ptt on --confirm-tx=<phrase> [--hold 3s] | ptt off")
	}
	unkey := func() error {
		return confirmCATUnkeyed(c, func() error { return c.SetPTT(false) }, 5*time.Second)
	}
	switch args[0] {
	case "off":
		return unkey()
	case "on":
		hold := parseDuration(a.flag("hold", "3s"), 3*time.Second)
		dec := safety.Check(a.flag("confirm-tx", ""))
		if dec.DryRun {
			fmt.Printf("[dry-run] would send: TX; ... (hold %s) ... RX;\n", hold)
			fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually key the transmitter.")
			return nil
		}
		if err := c.SetPTT(true); err != nil {
			return err
		}
		fmt.Printf("PTT ON — auto-unkeying after %s\n", hold)
		time.Sleep(hold)
		if err := unkey(); err != nil {
			return fmt.Errorf("PTT ON succeeded but could not confirm unkey: %w", err)
		}
		fmt.Println("PTT OFF (confirmed)")
		return nil
	default:
		return fmt.Errorf("ptt: unknown value %q (want on|off)", args[0])
	}
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
