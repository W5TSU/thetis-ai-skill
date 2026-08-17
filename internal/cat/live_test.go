//go:build live

// Live integration tests against a real, running Thetis instance over CAT.
// Excluded from normal `go test ./...` and CI by the "live" build tag —
// run explicitly with:
//
//	THETIS_HOST=192.168.2.12 go test -tags=live ./internal/cat/... -v
//
// Every test here round-trips a setting (read original, change it, verify,
// restore the original) rather than asserting a specific value, since the
// live radio's actual state is unknown and shouldn't be assumed. Nothing in
// this file ever keys the transmitter — SetPTT is only ever called with
// false, which is always safe (it either confirms RX is already active or
// defensively unkeys). Testing SetPTT(true) requires the same
// human-in-the-loop confirmation as `thetisctl cat ptt on --confirm-tx=...`
// and belongs in a manual run, not an automated test — see
// .claude/skills/thetis-control/SKILL.md's safety protocol.
package cat

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	host := os.Getenv("THETIS_HOST")
	if host == "" {
		t.Skip("set THETIS_HOST (e.g. THETIS_HOST=192.168.2.12) to run live tests against a real Thetis instance")
	}
	port := os.Getenv("THETIS_CAT_PORT")
	if port == "" {
		port = "13013"
	}
	timeout := 5 * time.Second
	if v := os.Getenv("THETIS_LIVE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}

	c, err := Dial(host+":"+port, timeout)
	if err != nil {
		t.Fatalf("Dial(%s:%s): %v", host, port, err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// TestLiveIdentity exercises GetID and GetIF — pure reads, safe to run
// unconditionally.
func TestLiveIdentity(t *testing.T) {
	c := liveClient(t)

	id, err := c.GetID()
	if err != nil {
		t.Fatalf("GetID: %v", err)
	}
	if id == "" {
		t.Error("GetID returned empty string")
	}
	t.Logf("rig ID: %s", id)

	st, err := c.GetIF()
	if err != nil {
		t.Fatalf("GetIF: %v", err)
	}
	t.Logf("IF status: %+v", st)
	if st.FreqHz == 0 {
		t.Error("GetIF returned 0 Hz — likely a parse bug, not a real radio state")
	}
}

// TestLiveFreqRoundTrip exercises SetVFOFreqHz/GetVFOFreqHz for VFO A.
func TestLiveFreqRoundTrip(t *testing.T) {
	c := liveClient(t)

	orig, err := c.GetVFOFreqHz("A")
	if err != nil {
		t.Fatalf("GetVFOFreqHz(A): %v", err)
	}
	t.Cleanup(func() {
		if err := c.SetVFOFreqHz("A", orig); err != nil {
			t.Errorf("restore VFO A to %d Hz: %v", orig, err)
		}
	})

	testFreq := orig + 1000 // 1 kHz up, well within any band
	if err := c.SetVFOFreqHz("A", testFreq); err != nil {
		t.Fatalf("SetVFOFreqHz(A, %d): %v", testFreq, err)
	}
	got, err := c.GetVFOFreqHz("A")
	if err != nil {
		t.Fatalf("GetVFOFreqHz(A) after set: %v", err)
	}
	if got != testFreq {
		t.Errorf("VFO A = %d Hz, want %d Hz", got, testFreq)
	}
}

// TestLiveModeRoundTrip exercises SetMode/GetMode.
func TestLiveModeRoundTrip(t *testing.T) {
	c := liveClient(t)

	orig, err := c.GetMode()
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	t.Cleanup(func() {
		if err := c.SetMode(orig); err != nil {
			t.Errorf("restore mode to %s: %v", orig, err)
		}
	})

	testMode := "USB"
	if orig == "USB" {
		testMode = "LSB"
	}
	if err := c.SetMode(testMode); err != nil {
		t.Fatalf("SetMode(%s): %v", testMode, err)
	}
	got, err := c.GetMode()
	if err != nil {
		t.Fatalf("GetMode after set: %v", err)
	}
	if got != testMode {
		t.Errorf("mode = %s, want %s", got, testMode)
	}
}

// TestLiveRITRoundTrip exercises SetRIT/GetRIT.
func TestLiveRITRoundTrip(t *testing.T) {
	c := liveClient(t)
	testBoolRoundTrip(t, "RIT", c.GetRIT, c.SetRIT)
}

// TestLiveXITRoundTrip exercises SetXIT/GetXIT.
func TestLiveXITRoundTrip(t *testing.T) {
	c := liveClient(t)
	testBoolRoundTrip(t, "XIT", c.GetXIT, c.SetXIT)
}

// TestLiveSplitRoundTrip exercises SetSplit/GetSplit.
func TestLiveSplitRoundTrip(t *testing.T) {
	c := liveClient(t)
	testBoolRoundTrip(t, "split", c.GetSplit, c.SetSplit)
}

// TestLivePowerRoundTrip exercises SetPowerOn/GetPowerOn — this is a bigger
// action than the other bool round trips (it actually starts/stops Thetis's
// HPSDR hardware connection and DSP audio engine, not just a UI flag), but
// it's still fully reversible and the test restores the original state via
// t.Cleanup, same as the others.
func TestLivePowerRoundTrip(t *testing.T) {
	c := liveClient(t)
	testBoolRoundTrip(t, "power", c.GetPowerOn, c.SetPowerOn)
}

func testBoolRoundTrip(t *testing.T, name string, get func() (bool, error), set func(bool) error) {
	t.Helper()

	orig, err := get()
	if err != nil {
		t.Fatalf("Get%s: %v", name, err)
	}
	t.Cleanup(func() {
		if err := set(orig); err != nil {
			t.Errorf("restore %s to %v: %v", name, orig, err)
		}
	})

	if err := set(!orig); err != nil {
		t.Fatalf("Set%s(%v): %v", name, !orig, err)
	}
	got, err := get()
	if err != nil {
		t.Fatalf("Get%s after set: %v", name, err)
	}
	if got != !orig {
		t.Errorf("%s = %v, want %v", name, got, !orig)
	}
}

// TestLiveAGCRoundTrip exercises SetAGC/GetAGC.
func TestLiveAGCRoundTrip(t *testing.T) {
	c := liveClient(t)

	orig, err := c.GetAGC()
	if err != nil {
		t.Fatalf("GetAGC: %v", err)
	}
	t.Cleanup(func() {
		if err := c.SetAGC(orig); err != nil {
			t.Errorf("restore AGC to %s: %v", orig, err)
		}
	})

	testMode := "FAST"
	if orig == "FAST" {
		testMode = "SLOW"
	}
	if err := c.SetAGC(testMode); err != nil {
		t.Fatalf("SetAGC(%s): %v", testMode, err)
	}
	got, err := c.GetAGC()
	if err != nil {
		t.Fatalf("GetAGC after set: %v", err)
	}
	if got != testMode {
		t.Errorf("AGC = %s, want %s", got, testMode)
	}
}

// TestLiveAttenuatorRoundTrip exercises SetAttenuatorDB/GetAttenuatorDB.
// GetAttenuatorDB (CAT ZZRX) has been observed to hang indefinitely on a
// live, actively-receiving radio — see SKILL.md's "Gotchas" section;
// suspected UI-thread contention from an automatic overload-protection
// feature, not a thetisctl bug. If that's still happening, this test skips
// with a clear note instead of failing outright (a known Thetis-side issue
// shouldn't masquerade as thetisctl breakage); if Thetis ever stops
// exhibiting it, this test starts actually verifying the round trip.
func TestLiveAttenuatorRoundTrip(t *testing.T) {
	c := liveClient(t)

	orig, err := c.GetAttenuatorDB()
	if err != nil {
		t.Skipf("GetAttenuatorDB failed (known live-radio issue, see SKILL.md gotchas): %v", err)
	}
	t.Cleanup(func() {
		if err := c.SetAttenuatorDB(orig); err != nil {
			t.Errorf("restore attenuator to %d dB: %v", orig, err)
		}
	})

	testDB := orig + 1
	if testDB > 31 {
		testDB = orig - 1
	}
	if err := c.SetAttenuatorDB(testDB); err != nil {
		t.Fatalf("SetAttenuatorDB(%d): %v", testDB, err)
	}
	got, err := c.GetAttenuatorDB()
	if err != nil {
		t.Fatalf("GetAttenuatorDB after set: %v", err)
	}
	if got != testDB {
		t.Errorf("attenuator = %d dB, want %d dB", got, testDB)
	}
}

// TestLiveAttenuatorSetDoesNotHang exercises SetAttenuatorDB in isolation,
// independent of the GetAttenuatorDB hang above. Unlike Query-based reads,
// Client.Set never waits for a reply (CAT set commands are fire-and-forget
// over this connection), so it's worth confirming whether the underlying
// issue is specific to the getter or affects the whole ZZRX command. Cannot
// restore the original value afterward (that would require the broken
// getter), so this deliberately picks a mid-range, low-consequence value
// (10 dB) rather than nudging relative to an unknown current state.
func TestLiveAttenuatorSetDoesNotHang(t *testing.T) {
	c := liveClient(t)
	if err := c.SetAttenuatorDB(10); err != nil {
		t.Fatalf("SetAttenuatorDB(10): %v", err)
	}
}

// TestLivePreampRoundTrip exercises SetPreamp. There is no typed GetPreamp
// (Thetis's ZZPA has no documented get path used elsewhere in this client),
// so this reads the current level via the raw Query escape hatch instead,
// exercising that code path too.
func TestLivePreampRoundTrip(t *testing.T) {
	c := liveClient(t)

	origStr, err := c.Query("ZZPA")
	if err != nil {
		t.Fatalf("Query(ZZPA): %v", err)
	}
	orig, err := strconv.Atoi(origStr)
	if err != nil {
		t.Fatalf("Query(ZZPA) returned non-numeric level %q: %v", origStr, err)
	}
	t.Cleanup(func() {
		if err := c.SetPreamp(orig); err != nil {
			t.Errorf("restore preamp to %d: %v", orig, err)
		}
	})

	testLevel := 1
	if orig == 1 {
		testLevel = 0
	}
	if err := c.SetPreamp(testLevel); err != nil {
		t.Fatalf("SetPreamp(%d): %v", testLevel, err)
	}
	gotStr, err := c.Query("ZZPA")
	if err != nil {
		t.Fatalf("Query(ZZPA) after set: %v", err)
	}
	got, err := strconv.Atoi(gotStr)
	if err != nil {
		t.Fatalf("Query(ZZPA) after set returned non-numeric level %q: %v", gotStr, err)
	}
	if got != testLevel {
		t.Errorf("preamp = %d, want %d", got, testLevel)
	}
}

// TestLiveBandGet exercises GetBand only. SetBand is deliberately not
// round-tripped automatically: unlike the other settings here, changing
// band can retune the VFO to that band's stored frequency
// (console.SetCATBand → GetBand/CATCommands.cs), which is a bigger and less
// predictable disruption to a live radio than the reversible toggles above.
// Verify SetBand manually (`thetisctl cat band set ...`) if needed.
func TestLiveBandGet(t *testing.T) {
	c := liveClient(t)

	band, err := c.GetBand()
	if err != nil {
		t.Fatalf("GetBand: %v", err)
	}
	if band == "" {
		t.Error("GetBand returned empty string")
	}
	t.Logf("band: %s", band)
}

// TestLivePTTOffIsSafe exercises SetPTT(false) only — confirming RX/unkey
// works — and confirms via GetIF that TX is not active afterward. SetPTT is
// never called with true here: that requires the same explicit,
// per-invocation human confirmation as the CLI's --confirm-tx flag, which an
// unattended test run cannot provide.
func TestLivePTTOffIsSafe(t *testing.T) {
	c := liveClient(t)

	if err := c.SetPTT(false); err != nil {
		t.Fatalf("SetPTT(false): %v", err)
	}
	st, err := c.GetIF()
	if err != nil {
		t.Fatalf("GetIF: %v", err)
	}
	if st.TXActive {
		t.Error("TX is active after SetPTT(false) — radio may be keyed by another source")
	}
}
