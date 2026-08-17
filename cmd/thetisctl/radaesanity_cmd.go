package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"thetisctl/internal/cat"
	"thetisctl/internal/safety"
)

// catRadaeSanity scripts the RADE V1 "off-air sanity check through the real
// Thetis pipeline" that FreeDV-Plan.md's Stage C notes as the next real test
// once RXRadaeEnabled had a CAT toggle (ZZDW/ZZDZ, added same session as this
// file). It's the RADE analogue of the FreeDV 700E bench test documented in
// Tools/FreeDV/README.md: instead of pressing Play and eyeballing the Setup
// tab, this drives the same three pieces (radae decode on, Quick Play
// injecting a prepared I/Q wav, poll sync/SNR) over CAT and prints a summary.
//
// Prerequisite this command does NOT do for you: the injectable wav must
// already be staged as Music\Thetis\quickrecord\SDRQuickAudio.wav on the
// Windows box. FreeDV-Plan.md records that the existing off-air capture
// (Tools/FreeDV/offair_14236000_RADEV1_20260808.wav) is *not* raw I/Q — it's
// post-demod mono audio duplicated into both channels (Quick-Rec taps a
// different pipeline point than Quick-Play's IQ-injection point) — so it
// needs re-synthesizing into a genuine analytic-signal I/Q wav first, e.g.
// via Tools/FreeDV/make_fdv_test_iq.py's --input-wav mode, before this
// command's Quick Play step will exercise the real RX DSP chain correctly.
//
// Quick Play is TX-capable (see catQuickPlay's comment — MOX on Playback
// defaults on in this codebase), so this reuses the same --confirm-tx gate
// and dry-run shape.
func catRadaeSanity(c *cat.Client, args []string, a parsedArgs) error {
	if len(args) != 0 {
		return fmt.Errorf("radae-sanity: takes no positional args, only flags — see 'thetisctl help'")
	}

	hold := parseDuration(a.flag("hold", "140s"), 140*time.Second)
	poll := parseDuration(a.flag("poll", "1s"), 1*time.Second)
	freqStr := a.flag("freq", "")
	mode := a.flag("mode", "")
	csvPath := a.flag("csv", "")

	dec := safety.Check(a.flag("confirm-tx", ""))
	if dec.DryRun {
		fmt.Println("[dry-run] would run the RADE off-air sanity check:")
		if freqStr != "" {
			fmt.Printf("  tune VFO A to %s Hz\n", freqStr)
		}
		if mode != "" {
			fmt.Printf("  set mode %s\n", mode)
		}
		fmt.Println("  radae on")
		fmt.Println("  quickplay on  <-- TX-capable: keys MOX if \"MOX on Playback\" is enabled")
		fmt.Printf("  poll radae status every %s for %s, logging sync/SNR\n", poll, hold)
		fmt.Println("  quickplay off (confirmed unkeyed)")
		fmt.Println("  radae off")
		fmt.Println("WARNING: this may key MOX for real, same caveat as 'quickplay on' — see its help.")
		fmt.Println("Pass --confirm-tx=" + safety.ConfirmPhrase + " to actually run it.")
		return nil
	}

	if freqStr != "" {
		hz, err := strconv.ParseUint(freqStr, 10, 64)
		if err != nil {
			return fmt.Errorf("radae-sanity: invalid --freq %q: %w", freqStr, err)
		}
		if err := c.SetVFOFreqHz("A", hz); err != nil {
			return fmt.Errorf("radae-sanity: tune VFO A: %w", err)
		}
		fmt.Printf("VFO A set to %d Hz\n", hz)
	}
	if mode != "" {
		if err := c.SetMode(mode); err != nil {
			return fmt.Errorf("radae-sanity: set mode: %w", err)
		}
		fmt.Println("Mode set to:", mode)
	}

	if err := c.SetRadaeDecode(true); err != nil {
		return fmt.Errorf("radae-sanity: enable radae decode: %w", err)
	}
	on, err := c.GetRadaeDecode()
	if err != nil || !on {
		return fmt.Errorf("radae-sanity: radae decode did not confirm enabled (on=%v, err=%v)", on, err)
	}
	fmt.Println("radae: enabled (confirmed)")

	// Cleanup runs even if the poll loop below errors partway through — never
	// leave Quick Play or the radae decoder running unattended.
	cleanup := func() {
		if err := confirmCATUnkeyed(c, func() error { return c.SetQuickPlay(false) }, 5*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, "radae-sanity: WARNING: could not confirm quickplay stopped:", err)
		} else {
			fmt.Println("quickplay: false (confirmed)")
		}
		if err := c.SetRadaeDecode(false); err != nil {
			fmt.Fprintln(os.Stderr, "radae-sanity: WARNING: could not disable radae decode:", err)
		} else {
			fmt.Println("radae: disabled")
		}
	}

	if err := c.SetQuickPlay(true); err != nil {
		cleanup()
		return fmt.Errorf("radae-sanity: start quickplay: %w", err)
	}
	fmt.Printf("quickplay: true — polling radae status every %s for %s\n", poll, hold)

	var csvFile *os.File
	if csvPath != "" {
		csvFile, err = os.Create(csvPath)
		if err != nil {
			cleanup()
			return fmt.Errorf("radae-sanity: create --csv file: %w", err)
		}
		defer csvFile.Close()
		fmt.Fprintln(csvFile, "elapsed_s,sync,snr_db")
	}

	var (
		polls      int
		errPolls   int
		syncedPoll int
		everSynced bool
		firstSync  time.Duration
		maxSNR     int
	)
	start := time.Now()
	deadline := start.Add(hold)
	for time.Now().Before(deadline) {
		elapsed := time.Since(start)
		st, err := c.GetRadaeStatus()
		polls++
		if err != nil {
			errPolls++
			fmt.Printf("[%6.1fs] error: %v\n", elapsed.Seconds(), err)
		} else {
			fmt.Printf("[%6.1fs] sync=%v snr=%ddB\n", elapsed.Seconds(), st.Sync, st.SNRdB)
			if csvFile != nil {
				fmt.Fprintf(csvFile, "%.1f,%d,%d\n", elapsed.Seconds(), boolToInt(st.Sync), st.SNRdB)
			}
			if st.Sync {
				syncedPoll++
				if !everSynced {
					everSynced = true
					firstSync = elapsed
				}
				if st.SNRdB > maxSNR {
					maxSNR = st.SNRdB
				}
			}
		}
		time.Sleep(poll)
	}

	cleanup()

	fmt.Println("--- radae-sanity summary ---")
	fmt.Printf("polls: %d (%d errored)\n", polls, errPolls)
	if everSynced {
		fmt.Printf("SYNC ACHIEVED: first at %.1fs, held for %d/%d polls (%.0f%%), max SNR %d dB\n",
			firstSync.Seconds(), syncedPoll, polls, 100*float64(syncedPoll)/float64(polls), maxSNR)
	} else {
		fmt.Println("no sync for the full run — see FreeDV-Plan.md Stage C for known open caveats")
		fmt.Println("(DNN decoder wiring status, and whether the injected wav is genuine analytic I/Q)")
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
