package tci

import (
	"math"
	"testing"
)

func TestStreamFrameRoundTrip(t *testing.T) {
	h := StreamHeader{
		ReceiverID: 1,
		SampleRate: 48000,
		SampleType: SampleInt16,
		Length:     3,
		StreamType: StreamTXAudio,
		Channels:   1,
	}
	samples := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}

	frame := BuildStreamFrame(h, samples)
	if len(frame) != streamHeaderSize+len(samples) {
		t.Fatalf("frame length = %d, want %d", len(frame), streamHeaderSize+len(samples))
	}

	gotHeader, gotSamples, err := ParseStreamFrame(frame)
	if err != nil {
		t.Fatalf("ParseStreamFrame: %v", err)
	}
	if gotHeader != h {
		t.Errorf("header = %+v, want %+v", gotHeader, h)
	}
	if string(gotSamples) != string(samples) {
		t.Errorf("samples = %v, want %v", gotSamples, samples)
	}
}

func TestParseStreamFrameTooShort(t *testing.T) {
	if _, _, err := ParseStreamFrame(make([]byte, 10)); err == nil {
		t.Fatal("ParseStreamFrame with 10-byte payload: want error, got nil")
	}
}

// TestEncodeDecodeRoundTrip mirrors Thetis's own encode/decode asymmetry
// (encode scales by signed-max-minus-one, decode divides by the power of
// two) — see TCIServer.cs encodeSamples/decodeSamples. Round-tripping through
// both must stay within about one quantization step (plus the deliberate
// scale mismatch) for integer types, and is exact for float32, which Thetis
// does not clip on encode.
func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		t    SampleType
		tol  float64
		clip bool
	}{
		{"int16", SampleInt16, 2.0 / 32767, true},
		{"int24", SampleInt24, 2.0 / 8388607, true},
		{"int32", SampleInt32, 2.0 / 2147483647, true},
		{"float32", SampleFloat32, 0, false},
	}
	input := []float32{0, 1, -1, 0.5, -0.5, 0.999, -0.999, 1.5, -1.5} // 1.5/-1.5 exercise int clipping

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded := EncodeSamples(input, c.t)
			decoded := DecodeSamples(encoded, c.t)
			if len(decoded) != len(input) {
				t.Fatalf("decoded length = %d, want %d", len(decoded), len(input))
			}
			for i, in := range input {
				want := in
				if c.clip {
					if want > 1 {
						want = 1
					} else if want < -1 {
						want = -1
					}
				}
				diff := math.Abs(float64(decoded[i] - want))
				if diff > c.tol+1e-6 {
					t.Errorf("sample %d: decoded %v, want ~%v (diff %v > tol %v)", i, decoded[i], want, diff, c.tol)
				}
			}
		})
	}
}

func TestEncodeSamplesEmpty(t *testing.T) {
	if got := EncodeSamples(nil, SampleInt16); got != nil {
		t.Errorf("EncodeSamples(nil) = %v, want nil", got)
	}
}

func TestWireNameRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		t    SampleType
		want string
	}{
		{SampleInt16, "int16"},
		{SampleInt24, "int24"},
		{SampleInt32, "int32"},
		{SampleFloat32, "float32"},
	} {
		if got := tc.t.WireName(); got != tc.want {
			t.Errorf("SampleType(%d).WireName() = %q, want %q", tc.t, got, tc.want)
		}
	}
}
