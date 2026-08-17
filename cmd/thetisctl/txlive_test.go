//go:build live

// Opt-in tests that actually key the transmitter for real, against a real
// Thetis instance. These are NOT part of the normal live test suite run
// (THETIS_HOST alone does not enable them) and must never be triggered
// automatically by an agent — per .claude/skills/thetis-control/SKILL.md's
// safety protocol, every real transmission needs a human operator's
// explicit, per-invocation go-ahead in the current conversation, which an
// automated test run cannot provide. This file exists so a human operator
// can run these deliberately themselves when they want full TX-path
// regression coverage, not so an agent can "complete the test suite" by
// running them unattended.
//
// To actually run these tests, set BOTH:
//
//	THETIS_HOST=<radio-ip>
//	THETIS_LIVE_ALLOW_TX=I-UNDERSTAND-THIS-KEYS-THE-RADIO   (exact phrase, see internal/safety.ConfirmPhrase)
//
// then:
//
//	go test -tags=live ./cmd/thetisctl/... -run TestLiveTX -v
//
// Every test here uses the smallest practical hold/duration and low
// power/speed settings, and the underlying CLI commands already guarantee
// an automatic unkey on completion, error, or Ctrl-C (see cat_cmd.go /
// tci_cmd.go's TX-capable command implementations).
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"thetisctl/internal/safety"
	"thetisctl/internal/tci"
)

// requireExplicitTXOptIn is the second, independent gate (on top of
// THETIS_HOST) that every test in this file must call before doing
// anything. Skips with a clear explanation if not set — this is the
// intended, expected outcome for any automated or unattended run.
func requireExplicitTXOptIn(t *testing.T) string {
	t.Helper()
	host := liveHost(t)
	if os.Getenv("THETIS_LIVE_ALLOW_TX") != safety.ConfirmPhrase {
		t.Skip("real-TX live tests require THETIS_LIVE_ALLOW_TX=" + safety.ConfirmPhrase +
			" in addition to THETIS_HOST — this is intentional; see this file's package doc comment. " +
			"Do not set this automatically; a human operator must opt in deliberately.")
	}
	return host
}

// TestLiveTXCatPTT keys CAT PTT for real for --hold's duration (kept short)
// and confirms the radio actually keys and then unkeys.
func TestLiveTXCatPTT(t *testing.T) {
	host := requireExplicitTXOptIn(t)
	output, err := captureStdout(t, func() error {
		return runCAT([]string{"--host", host, "ptt", "on",
			"--confirm-tx=" + safety.ConfirmPhrase, "--hold", "2s"})
	})
	if err != nil {
		t.Fatalf("runCAT(ptt on, confirmed): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "PTT ON") || !strings.Contains(output, "PTT OFF") {
		t.Errorf("expected to see both PTT ON and PTT OFF in output, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveTXTCITune keys TCI TUNE (bare carrier) for real for --hold's
// duration (kept short) and confirms it keys and then unkeys.
func TestLiveTXTCITune(t *testing.T) {
	host := requireExplicitTXOptIn(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "tune", "0", "on",
			"--confirm-tx=" + safety.ConfirmPhrase, "--hold", "2s"})
	})
	if err != nil {
		t.Fatalf("runTCI(tune 0 on, confirmed): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "TUNE ON") || !strings.Contains(output, "TUNE OFF") {
		t.Errorf("expected to see both TUNE ON and TUNE OFF in output, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveTXTCIPTT keys TCI PTT directly (no CW/audio source attached) for
// real for --hold's duration (kept short) and confirms it keys and unkeys.
func TestLiveTXTCIPTT(t *testing.T) {
	host := requireExplicitTXOptIn(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "ptt", "0", "on",
			"--confirm-tx=" + safety.ConfirmPhrase, "--hold", "2s"})
	})
	if err != nil {
		t.Fatalf("runTCI(ptt 0 on, confirmed): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "PTT ON") || !strings.Contains(output, "PTT OFF") {
		t.Errorf("expected to see both PTT ON and PTT OFF in output, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveTXTCICW sends a short real CW message and confirms it completes
// (mirrors the manual "TEST DE W5TSU" verification done earlier this
// session, as an automated regression test).
func TestLiveTXTCICW(t *testing.T) {
	host := requireExplicitTXOptIn(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "cw", "send", "0", "TEST",
			"--speed", "25", "--confirm-tx=" + safety.ConfirmPhrase, "--max-duration", "15s"})
	})
	if err != nil {
		t.Fatalf("runTCI(cw send, confirmed): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "CW done") {
		t.Errorf("expected to see CW completion in output, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveTXTCITxAudio streams a short, quiet real audio tone as TX audio
// and confirms it completes and unkeys.
func TestLiveTXTCITxAudio(t *testing.T) {
	host := requireExplicitTXOptIn(t)

	wavPath := filepath.Join(t.TempDir(), "tone.wav")
	samples := make([]float32, 48000/2) // 0.5s at 48kHz mono
	for i := range samples {
		samples[i] = 0.05 // quiet, well below clipping
	}
	if err := tci.WriteWAV(wavPath, tci.WAVFormat{SampleRate: 48000, Channels: 1, BitsPerSample: 16}, samples); err != nil {
		t.Fatalf("write test WAV: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "tx-audio", "send", "0", "--file", wavPath,
			"--confirm-tx=" + safety.ConfirmPhrase, "--max-duration", "3s"})
	})
	if err != nil {
		t.Fatalf("runTCI(tx-audio send, confirmed): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "TX ON") || !strings.Contains(output, "TX OFF") {
		t.Errorf("expected to see both TX ON and TX OFF in output, got: %s", output)
	}
	assertNotTransmitting(t, host)
}
