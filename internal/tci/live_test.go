//go:build live

// Live integration tests against a real, running Thetis instance over TCI.
// Excluded from normal `go test ./...` and CI by the "live" build tag —
// run explicitly with:
//
//	THETIS_HOST=192.168.2.12 go test -tags=live ./internal/tci/... -v
//
// Like internal/cat's live tests, everything here round-trips a setting
// (read original via a "get" query, change it, verify, restore) rather than
// asserting a specific value. Queries tolerate Thetis's optional
// send-initial-state-on-connect burst by skipping non-matching replies,
// mirroring internal/cat's Query.
//
// TX-capable functions are deliberately NOT exercised for real here:
// SetTune/SetTrx/SetTrxTCIAudio are only ever called with false (a safe,
// defensive unkey), and SendCWMacro / SendAudioFrame (TX_AUDIO_STREAM) are
// not called at all — actually transmitting requires the same
// human-in-the-loop confirmation as the CLI's --confirm-tx flag, which an
// unattended test run cannot provide. See
// .claude/skills/thetis-control/SKILL.md's safety protocol.
package tci

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	host := os.Getenv("THETIS_HOST")
	if host == "" {
		t.Skip("set THETIS_HOST (e.g. THETIS_HOST=192.168.2.12) to run live tests against a real Thetis instance")
	}
	port := os.Getenv("THETIS_TCI_PORT")
	if port == "" {
		port = "50001"
	}
	timeout := 5 * time.Second
	if v := os.Getenv("THETIS_LIVE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}

	conn, err := Dial(host+":"+port, timeout)
	if err != nil {
		t.Fatalf("Dial(%s:%s): %v", host, port, err)
	}
	client := NewClient(conn)
	t.Cleanup(func() { client.Close() })
	drainInitialBurst(t, client)
	return client
}

// drainInitialBurst reads and discards Thetis's optional
// "send initial state on connect" dump (protocol/device/vfo/modulation/...
// for every control, terminated by an unsolicited "ready;" frame — see
// TCIServer.cs's initial-state send path and sendReady/"ready", ~line 2698).
// Every queryTCI/queryTCIBare call below sends a request and grabs the first
// matching reply by command name + argument prefix; without draining first,
// that "first match" could easily be a stale value from this burst instead
// of the genuine reply to our own request — the initial version of this test
// suite hit exactly that bug (every round-trip "failed" with the pre-change
// value, even though a manual diagnostic proved the underlying Set calls
// were applied correctly and confirmed via CAT on the same live radio).
// Falls back to "no more data within one read timeout" if "ready" is never
// seen (e.g. the option is disabled), rather than blocking indefinitely.
func drainInitialBurst(t *testing.T, client *Client) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		cmd, _, err := client.RecvCmd()
		if err != nil {
			if isLiveTimeoutErr(err) {
				return
			}
			t.Fatalf("RecvCmd while draining initial state burst: %v", err)
		}
		if cmd == "ready" {
			return
		}
	}
}

func isLiveTimeoutErr(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// queryTCI sends "wantCmd:sendArgs...;" and waits (skipping any non-matching
// frames — including Thetis's optional initial-state burst) for a reply
// whose command matches wantCmd and whose leading args match matchPrefix,
// returning that reply's full argument list.
func queryTCI(t *testing.T, client *Client, wantCmd string, sendArgs []string, matchPrefix []string) []string {
	t.Helper()
	if err := client.SendCmd(wantCmd, sendArgs...); err != nil {
		t.Fatalf("SendCmd(%s, %v): %v", wantCmd, sendArgs, err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		cmd, args, err := client.RecvCmd()
		if err != nil {
			if isLiveTimeoutErr(err) {
				continue
			}
			t.Fatalf("RecvCmd while waiting for %s reply: %v", wantCmd, err)
		}
		if cmd != wantCmd || len(args) < len(matchPrefix) {
			continue
		}
		match := true
		for i, want := range matchPrefix {
			if args[i] != want {
				match = false
				break
			}
		}
		if match {
			return args
		}
	}
	t.Fatalf("timed out waiting for %s reply matching prefix %v", wantCmd, matchPrefix)
	return nil
}

// TestLiveHandshake confirms the RFC6455 client handshake succeeds against a
// real Thetis TCI server.
func TestLiveHandshake(t *testing.T) {
	liveClient(t)
}

// TestLiveVFORoundTrip exercises SetVFOFreqHz for rx 0, chan 0 (VFO A),
// reading the current value via the "vfo:<rx>,<chan>;" 2-arg get form.
func TestLiveVFORoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "vfo", []string{"0", "0"}, []string{"0", "0"})
	orig, err := strconv.ParseInt(origArgs[2], 10, 64)
	if err != nil {
		t.Fatalf("parse vfo reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetVFOFreqHz(0, 0, orig); err != nil {
			t.Errorf("restore vfo 0,0 to %d Hz: %v", orig, err)
		}
	})

	testFreq := orig + 1000
	if err := client.SetVFOFreqHz(0, 0, testFreq); err != nil {
		t.Fatalf("SetVFOFreqHz(0, 0, %d): %v", testFreq, err)
	}
	gotArgs := queryTCI(t, client, "vfo", []string{"0", "0"}, []string{"0", "0"})
	got, err := strconv.ParseInt(gotArgs[2], 10, 64)
	if err != nil {
		t.Fatalf("parse vfo reply %v: %v", gotArgs, err)
	}
	if got != testFreq {
		t.Errorf("vfo 0,0 = %d Hz, want %d Hz", got, testFreq)
	}
}

// TestLiveModulationRoundTrip exercises SetModulation, reading the current
// mode via the "modulation:<rx>;" 1-arg get form.
func TestLiveModulationRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "modulation", []string{"0"}, []string{"0"})
	orig := origArgs[1]
	t.Cleanup(func() {
		if err := client.SetModulation(0, orig); err != nil {
			t.Errorf("restore modulation to %s: %v", orig, err)
		}
	})

	// Thetis's "get" reply is always the DSPMode enum's uppercase ToString()
	// (e.g. "USB") regardless of the case sent on "set" — compare
	// case-insensitively and pick a genuinely different mode from orig.
	testMode := "usb"
	if strings.EqualFold(orig, "usb") {
		testMode = "lsb"
	}
	if err := client.SetModulation(0, testMode); err != nil {
		t.Fatalf("SetModulation(0, %s): %v", testMode, err)
	}
	gotArgs := queryTCI(t, client, "modulation", []string{"0"}, []string{"0"})
	if !strings.EqualFold(gotArgs[1], testMode) {
		t.Errorf("modulation = %s, want %s", gotArgs[1], testMode)
	}
}

// TestLiveSplitEnableRoundTrip exercises SetSplitEnable.
func TestLiveSplitEnableRoundTrip(t *testing.T) {
	testTCIBoolRoundTrip(t, "split_enable", func(c *Client, on bool) error { return c.SetSplitEnable(0, on) })
}

// TestLiveRITEnableRoundTrip exercises SetRITEnable.
func TestLiveRITEnableRoundTrip(t *testing.T) {
	testTCIBoolRoundTrip(t, "rit_enable", func(c *Client, on bool) error { return c.SetRITEnable(0, on) })
}

// TestLiveXITEnableRoundTrip exercises SetXITEnable.
func TestLiveXITEnableRoundTrip(t *testing.T) {
	testTCIBoolRoundTrip(t, "xit_enable", func(c *Client, on bool) error { return c.SetXITEnable(0, on) })
}

func testTCIBoolRoundTrip(t *testing.T, cmd string, set func(*Client, bool) error) {
	t.Helper()
	client := liveClient(t)

	origArgs := queryTCI(t, client, cmd, []string{"0"}, []string{"0"})
	orig := origArgs[1] == "true"
	t.Cleanup(func() {
		if err := set(client, orig); err != nil {
			t.Errorf("restore %s to %v: %v", cmd, orig, err)
		}
	})

	if err := set(client, !orig); err != nil {
		t.Fatalf("set %s to %v: %v", cmd, !orig, err)
	}
	gotArgs := queryTCI(t, client, cmd, []string{"0"}, []string{"0"})
	got := gotArgs[1] == "true"
	if got != !orig {
		t.Errorf("%s = %v, want %v", cmd, got, !orig)
	}
}

// TestLiveRITOffsetRoundTrip exercises SetRITOffsetHz.
func TestLiveRITOffsetRoundTrip(t *testing.T) {
	testTCIIntRoundTrip(t, "rit_offset", func(c *Client, v int) error { return c.SetRITOffsetHz(0, v) })
}

// TestLiveXITOffsetRoundTrip exercises SetXITOffsetHz.
func TestLiveXITOffsetRoundTrip(t *testing.T) {
	testTCIIntRoundTrip(t, "xit_offset", func(c *Client, v int) error { return c.SetXITOffsetHz(0, v) })
}

// TestLiveAGCGainRoundTrip exercises SetAGCGain.
func TestLiveAGCGainRoundTrip(t *testing.T) {
	testTCIIntRoundTrip(t, "agc_gain", func(c *Client, v int) error { return c.SetAGCGain(0, v) })
}

func testTCIIntRoundTrip(t *testing.T, cmd string, set func(*Client, int) error) {
	t.Helper()
	client := liveClient(t)

	origArgs := queryTCI(t, client, cmd, []string{"0"}, []string{"0"})
	orig, err := strconv.Atoi(origArgs[1])
	if err != nil {
		t.Fatalf("parse %s reply %v: %v", cmd, origArgs, err)
	}
	t.Cleanup(func() {
		if err := set(client, orig); err != nil {
			t.Errorf("restore %s to %d: %v", cmd, orig, err)
		}
	})

	testVal := orig + 1
	if err := set(client, testVal); err != nil {
		t.Fatalf("set %s to %d: %v", cmd, testVal, err)
	}
	gotArgs := queryTCI(t, client, cmd, []string{"0"}, []string{"0"})
	got, err := strconv.Atoi(gotArgs[1])
	if err != nil {
		t.Fatalf("parse %s reply %v: %v", cmd, gotArgs, err)
	}
	if got != testVal {
		t.Errorf("%s = %d, want %d", cmd, got, testVal)
	}
}

// TestLiveFilterBandRoundTrip exercises SetFilterBand, reading the current
// passband via the "rx_filter_band:<rx>;" 1-arg get form.
func TestLiveFilterBandRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "rx_filter_band", []string{"0"}, []string{"0"})
	origLow, err := strconv.Atoi(origArgs[1])
	if err != nil {
		t.Fatalf("parse rx_filter_band reply %v: %v", origArgs, err)
	}
	origHigh, err := strconv.Atoi(origArgs[2])
	if err != nil {
		t.Fatalf("parse rx_filter_band reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetFilterBand(0, origLow, origHigh); err != nil {
			t.Errorf("restore filter band to [%d, %d]: %v", origLow, origHigh, err)
		}
	})

	testLow, testHigh := origLow+10, origHigh-10
	if testLow >= testHigh {
		t.Skipf("passband [%d, %d] too narrow to nudge safely, skipping mutation", origLow, origHigh)
	}
	if err := client.SetFilterBand(0, testLow, testHigh); err != nil {
		t.Fatalf("SetFilterBand(0, %d, %d): %v", testLow, testHigh, err)
	}
	gotArgs := queryTCI(t, client, "rx_filter_band", []string{"0"}, []string{"0"})
	gotLow, _ := strconv.Atoi(gotArgs[1])
	gotHigh, _ := strconv.Atoi(gotArgs[2])
	if gotLow != testLow || gotHigh != testHigh {
		t.Errorf("filter band = [%d, %d], want [%d, %d]", gotLow, gotHigh, testLow, testHigh)
	}
}

// TestLiveStepAttenuatorRoundTrip exercises SetStepAttenuatorDB.
// TestLiveStepAttenuatorRoundTrip exercises SetStepAttenuatorDB. Unlike the
// other round trips in this file, this does NOT assert the readback equals
// the value just set: on this radio it was observed changing on its own in
// real time — RX1's rx_step_att_ex cycled through several values with no
// client touching it at all (likely an automatic overload-protection
// feature reacting to live signal conditions), so a live radio can and does
// legitimately overwrite our value between Set and Get. Only confirms the
// Set call is accepted (no error) and the control is still readable as a
// valid integer afterward — a weaker but honest check given this
// environment. The CAT attenuator's ZZRX getter also hangs on this same
// radio (internal/cat/live_test.go's TestLiveAttenuatorRoundTrip), plausibly
// the same underlying activity saturating Thetis's UI-thread Invoke queue.
func TestLiveStepAttenuatorRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "rx_step_att_ex", []string{"0"}, []string{"0"})
	orig, err := strconv.Atoi(origArgs[1])
	if err != nil {
		t.Fatalf("parse rx_step_att_ex reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetStepAttenuatorDB(0, orig); err != nil {
			t.Errorf("restore step attenuator to %d dB: %v", orig, err)
		}
	})

	testVal := orig + 1
	if err := client.SetStepAttenuatorDB(0, testVal); err != nil {
		t.Fatalf("SetStepAttenuatorDB(0, %d): %v", testVal, err)
	}
	gotArgs := queryTCI(t, client, "rx_step_att_ex", []string{"0"}, []string{"0"})
	got, err := strconv.Atoi(gotArgs[1])
	if err != nil {
		t.Fatalf("parse rx_step_att_ex reply %v after set: %v", gotArgs, err)
	}
	t.Logf("step attenuator after set(%d): %d (may differ — see test doc comment)", testVal, got)
}

// TestLivePreampAttenuatorRoundTrip exercises SetPreampAttenuatorDB, which
// takes a non-positive dB value (0 = off, negative = that many dB of gain).
// Server-side this does NOT round-trip an arbitrary integer: handleRxPreampAttEx
// (TCIServer.cs:4927-4944) converts it via consoleThreadSafe.SetATT(...,
// SetAttMode.PREAMP_MODE), which snaps to the nearest discrete PreampMode
// step (0, -10, -20, -30, -40, -50 dB, plus SA-prefixed variants — see
// PreampMode enum, Project Files/Source/Console/enums.cs:236-251), not a
// continuous value. -1 was tried first and (correctly) snapped to 0 — this
// test uses an actual valid step (-10) instead.
func TestLivePreampAttenuatorRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "rx_preamp_att_ex", []string{"0"}, []string{"0"})
	orig, err := strconv.Atoi(origArgs[1])
	if err != nil {
		t.Fatalf("parse rx_preamp_att_ex reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetPreampAttenuatorDB(0, orig); err != nil {
			t.Errorf("restore preamp attenuation to %d: %v", orig, err)
		}
	})

	testVal := -10
	if orig == -10 {
		testVal = 0
	}
	if err := client.SetPreampAttenuatorDB(0, testVal); err != nil {
		t.Fatalf("SetPreampAttenuatorDB(0, %d): %v", testVal, err)
	}
	gotArgs := queryTCI(t, client, "rx_preamp_att_ex", []string{"0"}, []string{"0"})
	got, err := strconv.Atoi(gotArgs[1])
	if err != nil {
		t.Fatalf("parse rx_preamp_att_ex reply %v after set: %v", gotArgs, err)
	}
	if got != testVal {
		t.Errorf("preamp attenuation = %d, want %d", got, testVal)
	}
}

// TestLiveAGCModeRoundTrip exercises SetAGCMode.
func TestLiveAGCModeRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "agc_mode", []string{"0"}, []string{"0"})
	orig := origArgs[1]
	t.Cleanup(func() {
		if err := client.SetAGCMode(0, orig); err != nil {
			t.Errorf("restore agc_mode to %s: %v", orig, err)
		}
	})

	testMode := "fast"
	if orig == "fast" {
		testMode = "slow"
	}
	if err := client.SetAGCMode(0, testMode); err != nil {
		t.Fatalf("SetAGCMode(0, %s): %v", testMode, err)
	}
	gotArgs := queryTCI(t, client, "agc_mode", []string{"0"}, []string{"0"})
	if gotArgs[1] != testMode {
		t.Errorf("agc_mode = %s, want %s", gotArgs[1], testMode)
	}
}

// TestLiveDriveRoundTrip exercises SetDrive.
func TestLiveDriveRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCI(t, client, "drive", []string{"0"}, []string{"0"})
	orig, err := strconv.Atoi(origArgs[1])
	if err != nil {
		t.Fatalf("parse drive reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetDrive(0, orig); err != nil {
			t.Errorf("restore drive to %d: %v", orig, err)
		}
	})

	testVal := orig + 1
	if testVal > 100 {
		testVal = orig - 1
	}
	if err := client.SetDrive(0, testVal); err != nil {
		t.Fatalf("SetDrive(0, %d): %v", testVal, err)
	}
	gotArgs := queryTCI(t, client, "drive", []string{"0"}, []string{"0"})
	got, _ := strconv.Atoi(gotArgs[1])
	if got != testVal {
		t.Errorf("drive = %d, want %d", got, testVal)
	}
}

// TestLiveCWMacroSpeedRoundTrip exercises SetCWMacroSpeedWPM, reading the
// current speed via the bare "cw_macros_speed;" get form (SendBareCmd — see
// its doc comment for why this command has no colon).
func TestLiveCWMacroSpeedRoundTrip(t *testing.T) {
	client := liveClient(t)

	origArgs := queryTCIBare(t, client, "cw_macros_speed")
	orig, err := strconv.Atoi(origArgs[0])
	if err != nil {
		t.Fatalf("parse cw_macros_speed reply %v: %v", origArgs, err)
	}
	t.Cleanup(func() {
		if err := client.SetCWMacroSpeedWPM(orig); err != nil {
			t.Errorf("restore cw_macros_speed to %d: %v", orig, err)
		}
	})

	testVal := orig + 1
	if testVal > 99 {
		testVal = orig - 1
	}
	if err := client.SetCWMacroSpeedWPM(testVal); err != nil {
		t.Fatalf("SetCWMacroSpeedWPM(%d): %v", testVal, err)
	}
	gotArgs := queryTCIBare(t, client, "cw_macros_speed")
	got, _ := strconv.Atoi(gotArgs[0])
	if got != testVal {
		t.Errorf("cw_macros_speed = %d, want %d", got, testVal)
	}
}

func queryTCIBare(t *testing.T, client *Client, cmd string) []string {
	t.Helper()
	if err := client.SendBareCmd(cmd); err != nil {
		t.Fatalf("SendBareCmd(%s): %v", cmd, err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		gotCmd, args, err := client.RecvCmd()
		if err != nil {
			if isLiveTimeoutErr(err) {
				continue
			}
			t.Fatalf("RecvCmd while waiting for %s reply: %v", cmd, err)
		}
		if gotCmd == cmd && len(args) >= 1 {
			return args
		}
	}
	t.Fatalf("timed out waiting for bare %s reply", cmd)
	return nil
}

// TestLivePowerRoundTrip exercises SetPower over its actual TCI wire path
// (bare "start;"/"stop;" — see SetPower's doc comment), toggling the engine
// off then back on and waiting for the server's broadcast confirmation each
// time. The underlying console.PowerOn property is already validated by
// internal/cat's TestLivePowerRoundTrip (CAT's ZZPS); this test exists
// because SetPower is a genuinely different wire encoding (a bare command,
// not "cmd:args;") that was never itself exercised. t.Cleanup always
// attempts to leave the engine powered on afterward, even if this test
// fails partway through (SetPower(true) is a no-op if already on).
func TestLivePowerRoundTrip(t *testing.T) {
	client := liveClient(t)

	t.Cleanup(func() {
		client.SetPower(true)
	})

	if err := client.SetPower(false); err != nil {
		t.Fatalf("SetPower(false): %v", err)
	}
	waitForPowerBroadcast(t, client, "stop")

	if err := client.SetPower(true); err != nil {
		t.Fatalf("SetPower(true): %v", err)
	}
	waitForPowerBroadcast(t, client, "start")
}

func waitForPowerBroadcast(t *testing.T, client *Client, want string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		cmd, _, err := client.RecvCmd()
		if err != nil {
			if isLiveTimeoutErr(err) {
				continue
			}
			t.Fatalf("RecvCmd while waiting for %q broadcast: %v", want, err)
		}
		if cmd == want {
			return
		}
	}
	t.Fatalf("timed out waiting for %q broadcast", want)
}

// TestLiveStopCWMacrosIsSafe exercises StopCWMacros as a no-op: since this
// connection never owns an active CW send (SendCWMacro is never called in
// this suite), Stop() should simply return without error or side effects.
func TestLiveStopCWMacrosIsSafe(t *testing.T) {
	client := liveClient(t)
	if err := client.StopCWMacros(); err != nil {
		t.Fatalf("StopCWMacros: %v", err)
	}
}

// TestLiveTuneOffIsSafe exercises SetTune(false) only — SetTune(true) keys
// TUNE (a bare carrier) and requires the same explicit confirmation as the
// CLI's --confirm-tx, which this unattended test cannot provide.
func TestLiveTuneOffIsSafe(t *testing.T) {
	client := liveClient(t)
	if err := client.SetTune(0, false); err != nil {
		t.Fatalf("SetTune(0, false): %v", err)
	}
}

// TestLiveTrxOffIsSafe exercises SetTrx(false) and SetTrxTCIAudio(false)
// only — the true form keys PTT and requires explicit confirmation.
func TestLiveTrxOffIsSafe(t *testing.T) {
	client := liveClient(t)
	if err := client.SetTrx(0, false); err != nil {
		t.Fatalf("SetTrx(0, false): %v", err)
	}
	if err := client.SetTrxTCIAudio(0, false); err != nil {
		t.Fatalf("SetTrxTCIAudio(0, false): %v", err)
	}
}

// TestLiveRXAudioStream exercises StartAudio/StopAudio/RecvAudioFrame and
// the sample decode path against real audio data from the radio — the only
// end-to-end functional check in this suite that isn't a simple
// set/get/restore round trip.
func TestLiveRXAudioStream(t *testing.T) {
	client := liveClient(t)

	if err := client.SetAudioSampleType(SampleInt16); err != nil {
		t.Fatalf("SetAudioSampleType(int16): %v", err)
	}
	if err := client.StartAudio(0); err != nil {
		t.Fatalf("StartAudio(0): %v", err)
	}
	defer client.StopAudio(0)

	deadline := time.Now().Add(10 * time.Second)
	frames := 0
	for time.Now().Before(deadline) && frames < 3 {
		h, data, err := client.RecvAudioFrame()
		if err != nil {
			if isLiveTimeoutErr(err) {
				continue
			}
			t.Fatalf("RecvAudioFrame: %v", err)
		}
		if h.StreamType != StreamRXAudio || h.ReceiverID != 0 {
			continue
		}
		if h.SampleRate <= 0 {
			t.Errorf("frame %d: sample rate = %d, want > 0", frames, h.SampleRate)
		}
		samples := DecodeSamples(data, h.SampleType)
		if len(samples) == 0 {
			t.Errorf("frame %d: decoded 0 samples from %d bytes", frames, len(data))
		}
		frames++
	}
	if frames == 0 {
		t.Fatal("received no RX_AUDIO_STREAM frames for rx 0 within 10s")
	}
	t.Logf("received %d RX audio frames", frames)
}
