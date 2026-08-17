package tci

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestWAVRoundTrip(t *testing.T) {
	cases := []WAVFormat{
		{SampleRate: 48000, Channels: 1, BitsPerSample: 16, Float: false},
		{SampleRate: 48000, Channels: 2, BitsPerSample: 16, Float: false},
		{SampleRate: 44100, Channels: 1, BitsPerSample: 24, Float: false},
		{SampleRate: 48000, Channels: 1, BitsPerSample: 32, Float: false},
		{SampleRate: 48000, Channels: 2, BitsPerSample: 32, Float: true},
	}
	samples := []float32{0, 1, -1, 0.5, -0.5, 0.25, -0.75, 0.1}

	for i, f := range cases {
		f := f
		path := filepath.Join(t.TempDir(), "test.wav")
		if err := WriteWAV(path, f, samples); err != nil {
			t.Fatalf("case %d: WriteWAV: %v", i, err)
		}
		gotFormat, gotSamples, err := ReadWAV(path)
		if err != nil {
			t.Fatalf("case %d: ReadWAV: %v", i, err)
		}
		if gotFormat != f {
			t.Errorf("case %d: format = %+v, want %+v", i, gotFormat, f)
		}
		if len(gotSamples) != len(samples) {
			t.Fatalf("case %d: got %d samples, want %d", i, len(gotSamples), len(samples))
		}
		tol := 1.0 / 32767
		if f.BitsPerSample == 24 {
			tol = 1.0 / 8388607
		} else if f.BitsPerSample == 32 && !f.Float {
			tol = 1.0 / 2147483647
		} else if f.Float {
			tol = 0
		}
		for j, s := range samples {
			if diff := math.Abs(float64(gotSamples[j] - s)); diff > tol+1e-6 {
				t.Errorf("case %d sample %d: got %v, want %v (diff %v > tol %v)", i, j, gotSamples[j], s, diff, tol)
			}
		}
	}
}

func TestReadWAVRejectsNonRIFF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-wav.wav")
	if err := WriteWAV(path, WAVFormat{SampleRate: 8000, Channels: 1, BitsPerSample: 16}, nil); err != nil {
		t.Fatal(err)
	}
	// Corrupt the RIFF magic and confirm ReadWAV rejects it.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := ReadWAV(path); err == nil {
		t.Fatal("ReadWAV with corrupted RIFF header: want error, got nil")
	}
}
