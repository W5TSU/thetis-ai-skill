package tci

import (
	"encoding/binary"
	"fmt"
	"math"
)

// SampleType mirrors TCISampleType (TCIServer.cs ~352).
type SampleType uint32

const (
	SampleInt16   SampleType = 0
	SampleInt24   SampleType = 1
	SampleInt32   SampleType = 2
	SampleFloat32 SampleType = 3
)

// WireName returns TCI's lowercase wire name for the sample type
// (int16/int24/int32/float32), as accepted by audio_stream_sample_type.
func (t SampleType) WireName() string {
	switch t {
	case SampleInt16:
		return "int16"
	case SampleInt24:
		return "int24"
	case SampleInt32:
		return "int32"
	default:
		return "float32"
	}
}

func (t SampleType) bytesPerSample() int {
	switch t {
	case SampleInt16:
		return 2
	case SampleInt24:
		return 3
	default: // INT32, FLOAT32
		return 4
	}
}

// StreamType mirrors TCIStreamType (TCIServer.cs ~343-348).
type StreamType uint32

const (
	StreamIQ      StreamType = 0
	StreamRXAudio StreamType = 1
	StreamTXAudio StreamType = 2
)

// StreamHeader is the 64-byte header prefixing every TCI binary frame, as
// built by buildStreamPayload (TCIServer.cs:5645-5667). All fields are
// little-endian uint32; the remaining reserved words are always zero.
type StreamHeader struct {
	ReceiverID int
	SampleRate int
	SampleType SampleType
	Length     int // sample count (not byte count)
	StreamType StreamType
	Channels   int
}

const streamHeaderSize = 64

// BuildStreamFrame serializes a header + raw sample bytes into a TCI binary
// frame payload.
func BuildStreamFrame(h StreamHeader, samples []byte) []byte {
	buf := make([]byte, streamHeaderSize+len(samples))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(h.ReceiverID))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(h.SampleRate))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(h.SampleType))
	// buf[12:16], buf[16:20] reserved, left zero
	binary.LittleEndian.PutUint32(buf[20:24], uint32(h.Length))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(h.StreamType))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(h.Channels))
	// buf[32:64] (8 reserved words) left zero
	copy(buf[streamHeaderSize:], samples)
	return buf
}

// ParseStreamFrame splits a TCI binary frame payload into its header and raw
// sample bytes.
func ParseStreamFrame(payload []byte) (StreamHeader, []byte, error) {
	if len(payload) < streamHeaderSize {
		return StreamHeader{}, nil, fmt.Errorf("tci: stream frame too short: %d bytes", len(payload))
	}
	h := StreamHeader{
		ReceiverID: int(binary.LittleEndian.Uint32(payload[0:4])),
		SampleRate: int(binary.LittleEndian.Uint32(payload[4:8])),
		SampleType: SampleType(binary.LittleEndian.Uint32(payload[8:12])),
		Length:     int(binary.LittleEndian.Uint32(payload[20:24])),
		StreamType: StreamType(binary.LittleEndian.Uint32(payload[24:28])),
		Channels:   int(binary.LittleEndian.Uint32(payload[28:32])),
	}
	return h, payload[streamHeaderSize:], nil
}

// EncodeSamples converts float32 samples in [-1, 1] to wire bytes, matching
// Thetis's own encodeSamples (TCIServer.cs:5669-5710): clip to [-1,1], then
// scale by the signed-max-minus-one convention (32767 / 8388607 / int32max),
// little-endian; FLOAT32 is a raw 4-byte copy.
func EncodeSamples(samples []float32, t SampleType) []byte {
	if len(samples) == 0 {
		return nil
	}
	if t == SampleFloat32 {
		buf := make([]byte, len(samples)*4)
		for i, s := range samples {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(s))
		}
		return buf
	}

	bps := t.bytesPerSample()
	buf := make([]byte, len(samples)*bps)
	off := 0
	for _, s := range samples {
		c := s
		if c > 1 {
			c = 1
		} else if c < -1 {
			c = -1
		}
		switch t {
		case SampleInt16:
			v := int16(math.Round(float64(c) * 32767))
			binary.LittleEndian.PutUint16(buf[off:], uint16(v))
			off += 2
		case SampleInt24:
			v := int32(math.Round(float64(c) * 8388607))
			buf[off] = byte(v)
			buf[off+1] = byte(v >> 8)
			buf[off+2] = byte(v >> 16)
			off += 3
		case SampleInt32:
			v := int32(math.Round(float64(c) * 2147483647))
			binary.LittleEndian.PutUint32(buf[off:], uint32(v))
			off += 4
		}
	}
	return buf
}

// DecodeSamples converts wire bytes to float32 samples, matching Thetis's
// own decodeSamples (TCIServer.cs:5712-5742): divide by the power-of-two
// convention (32768 / 8388608 / 2147483648), little-endian; FLOAT32 is a raw
// 4-byte copy. Note the deliberate encode/decode scale mismatch (32767 vs
// 32768 etc.) matches Thetis's own asymmetric round-trip exactly.
func DecodeSamples(data []byte, t SampleType) []float32 {
	bps := t.bytesPerSample()
	if bps == 0 || len(data) < bps {
		return nil
	}
	n := len(data) / bps
	out := make([]float32, n)
	off := 0
	for i := 0; i < n; i++ {
		switch t {
		case SampleInt16:
			v := int16(binary.LittleEndian.Uint16(data[off:]))
			out[i] = float32(v) / 32768.0
			off += 2
		case SampleInt24:
			v := int32(data[off]) | int32(data[off+1])<<8 | int32(data[off+2])<<16
			if v&0x800000 != 0 {
				v |= ^int32(0xFFFFFF) // sign-extend
			}
			out[i] = float32(v) / 8388608.0
			off += 3
		case SampleInt32:
			v := int32(binary.LittleEndian.Uint32(data[off:]))
			out[i] = float32(v) / 2147483648.0
			off += 4
		case SampleFloat32:
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[off:]))
			off += 4
		}
	}
	return out
}
