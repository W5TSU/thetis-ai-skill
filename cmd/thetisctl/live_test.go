//go:build live

// Live integration tests against a real, running Thetis instance, exercised
// through the actual CLI entry points (runCAT/runTCI) rather than the
// internal/cat and internal/tci packages directly — internal/*/live_test.go
// already cover the client library; this file covers CLI-layer code (flag
// parsing, WAV file I/O, stdout streaming, dry-run gating) that a library
// call bypasses. Excluded from `go test ./...` and CI by the "live" build
// tag — run explicitly:
//
//	THETIS_HOST=192.168.2.12 go test -tags=live ./cmd/thetisctl/... -v
//
// TX-capable commands (ptt, tune, cw send, tx-audio send) are only ever
// exercised as dry runs here (--confirm-tx is never set) — see
// txlive_test.go for the separately-gated, opt-in real-TX test scaffolding
// that a human operator can run deliberately.
package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"thetisctl/internal/cat"
	"thetisctl/internal/tci"
)

func liveHost(t *testing.T) string {
	t.Helper()
	host := os.Getenv("THETIS_HOST")
	if host == "" {
		t.Skip("set THETIS_HOST (e.g. THETIS_HOST=192.168.2.12) to run live tests against a real Thetis instance")
	}
	return host
}

// captureStdout redirects the process's os.Stdout for the duration of fn,
// returning everything written to it. thetisctl's command functions print
// directly via fmt.Print*(os.Stdout) rather than an injectable io.Writer, so
// this is the only way to observe their output in-process.
func captureStdout(t *testing.T, fn func() error) (output string, err error) {
	t.Helper()
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	err = fn()

	os.Stdout = orig
	w.Close()
	output = <-done
	r.Close()
	return output, err
}

// assertNotTransmitting cross-checks over an independent CAT connection
// that the radio is not keyed — used after every dry-run test as a
// belt-and-suspenders confirmation that "dry run" really never transmitted.
func assertNotTransmitting(t *testing.T, host string) {
	t.Helper()
	c, err := cat.Dial(host+":13013", 5*time.Second)
	if err != nil {
		t.Fatalf("assertNotTransmitting: Dial CAT: %v", err)
	}
	defer c.Close()
	st, err := c.GetIF()
	if err != nil {
		t.Fatalf("assertNotTransmitting: GetIF: %v", err)
	}
	if st.TXActive {
		t.Fatal("radio is still transmitting — either a dry run keyed for real, or a real TX command's unkey didn't take effect")
	}
}

// TestLiveCLIRxAudioCaptureWritesValidWAV exercises the actual
// "tci rx-audio capture" CLI path end to end: flag parsing, StartAudio,
// receiving real frames, and WriteWAV — none of which internal/tci's
// TestLiveRXAudioStream touches (it calls the client functions directly).
func TestLiveCLIRxAudioCaptureWritesValidWAV(t *testing.T) {
	host := liveHost(t)
	out := filepath.Join(t.TempDir(), "capture.wav")

	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "rx-audio", "capture", "0", "--duration", "3s", "--out", out})
	})
	if err != nil {
		t.Fatalf("runTCI(rx-audio capture): %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "captured") {
		t.Errorf("expected a summary line containing %q, got: %s", "captured", output)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read captured WAV: %v", err)
	}
	if len(data) < 44 {
		t.Fatalf("captured file too short to be a valid WAV: %d bytes", len(data))
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Errorf("captured file is not a valid RIFF/WAVE: %q", data[0:12])
	}
}

// TestLiveCLIRxAudioStreamWritesRawPCM exercises "tci rx-audio stream",
// which writes raw float32 LE PCM straight to stdout instead of a WAV file
// — a code path TestLiveCLIRxAudioCaptureWritesValidWAV doesn't cover.
func TestLiveCLIRxAudioStreamWritesRawPCM(t *testing.T) {
	host := liveHost(t)

	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "rx-audio", "stream", "0", "--duration", "2s"})
	})
	if err != nil {
		t.Fatalf("runTCI(rx-audio stream): %v", err)
	}
	if len(output)%4 != 0 {
		t.Errorf("raw PCM output length %d is not a multiple of 4 bytes (float32 samples)", len(output))
	}
	if len(output) == 0 {
		t.Fatal("received no PCM data")
	}
	t.Logf("received %d bytes (%d float32 samples)", len(output), len(output)/4)
}

// TestLiveCLIQuery exercises the "tci query" raw passthrough command.
// Originally caught a real bug here: tciQuery grabbed the very first reply
// frame without matching it against the request, so querying right after
// connecting (while Thetis's initial-state burst was still arriving) could
// return an unrelated burst frame instead of the genuine reply — fixed in
// tciQuery to match by command name (see its doc comment for the residual
// ambiguity that fix can't fully resolve).
func TestLiveCLIQuery(t *testing.T) {
	host := liveHost(t)

	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "query", "vfo", "0", "0"})
	})
	if err != nil {
		t.Fatalf("runTCI(query vfo 0 0): %v\noutput: %s", err, output)
	}
	t.Logf("query output: %s", strings.TrimSpace(output))
	if !strings.HasPrefix(output, "vfo:") {
		t.Errorf("query vfo 0 0: expected output starting with %q, got %q", "vfo:", output)
	}
}

// TestLiveCLIDryRunCatPTT confirms "cat ptt on" without --confirm-tx prints
// a dry-run notice, succeeds, and never keys the transmitter.
func TestLiveCLIDryRunCatPTT(t *testing.T) {
	host := liveHost(t)
	output, err := captureStdout(t, func() error {
		return runCAT([]string{"--host", host, "ptt", "on"})
	})
	if err != nil {
		t.Fatalf("runCAT(ptt on) dry run: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run notice, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveCLIDryRunTCITune confirms "tci tune 0 on" without --confirm-tx
// prints a dry-run notice, succeeds, and never keys the transmitter.
func TestLiveCLIDryRunTCITune(t *testing.T) {
	host := liveHost(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "tune", "0", "on"})
	})
	if err != nil {
		t.Fatalf("runTCI(tune 0 on) dry run: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run notice, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveCLIDryRunTCIPTT confirms "tci ptt 0 on" without --confirm-tx
// prints a dry-run notice, succeeds, and never keys the transmitter.
func TestLiveCLIDryRunTCIPTT(t *testing.T) {
	host := liveHost(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "ptt", "0", "on"})
	})
	if err != nil {
		t.Fatalf("runTCI(ptt 0 on) dry run: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run notice, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveCLIDryRunTCICW confirms "tci cw send" without --confirm-tx prints
// a dry-run notice, succeeds, and never keys the transmitter.
func TestLiveCLIDryRunTCICW(t *testing.T) {
	host := liveHost(t)
	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "cw", "send", "0", "TEST", "--speed", "20"})
	})
	if err != nil {
		t.Fatalf("runTCI(cw send) dry run: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run notice, got: %s", output)
	}
	assertNotTransmitting(t, host)
}

// TestLiveCLIDryRunTCITxAudio confirms "tci tx-audio send" without
// --confirm-tx prints a dry-run notice, succeeds, and never keys the
// transmitter — exercising WAV file reading (tci.ReadWAV) along the way,
// a code path no other test in this suite touches.
func TestLiveCLIDryRunTCITxAudio(t *testing.T) {
	host := liveHost(t)

	wavPath := filepath.Join(t.TempDir(), "tone.wav")
	samples := make([]float32, 4800) // 0.1s of silence at 48kHz mono — plenty for a dry run
	if err := tci.WriteWAV(wavPath, tci.WAVFormat{SampleRate: 48000, Channels: 1, BitsPerSample: 16}, samples); err != nil {
		t.Fatalf("write test WAV: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runTCI([]string{"--host", host, "tx-audio", "send", "0", "--file", wavPath})
	})
	if err != nil {
		t.Fatalf("runTCI(tx-audio send) dry run: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run notice, got: %s", output)
	}
	assertNotTransmitting(t, host)
}
