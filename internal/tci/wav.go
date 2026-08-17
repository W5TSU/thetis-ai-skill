package tci

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// WAVFormat describes a PCM or IEEE-float WAV file's format chunk.
type WAVFormat struct {
	SampleRate    int
	Channels      int
	BitsPerSample int  // 16, 24, or 32
	Float         bool // true = 32-bit IEEE float (BitsPerSample must be 32)
}

const (
	wavFormatPCM   = 1
	wavFormatFloat = 3
)

// ReadWAV reads a canonical little-endian RIFF/WAVE file (PCM 16/24/32-bit
// int, or 32-bit IEEE float) and returns its format and samples as
// interleaved float32 in [-1, 1].
func ReadWAV(path string) (WAVFormat, []float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WAVFormat{}, nil, fmt.Errorf("tci: read %s: %w", path, err)
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return WAVFormat{}, nil, fmt.Errorf("tci: %s is not a RIFF/WAVE file", path)
	}

	var format WAVFormat
	var audioFormat uint16
	var haveFmt bool
	var sampleBytes []byte

	pos := 12
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		body := pos + 8
		if body+size > len(data) {
			size = len(data) - body // tolerate a truncated final chunk
		}

		switch id {
		case "fmt ":
			if size < 16 {
				return WAVFormat{}, nil, fmt.Errorf("tci: %s: fmt chunk too short", path)
			}
			audioFormat = binary.LittleEndian.Uint16(data[body : body+2])
			format.Channels = int(binary.LittleEndian.Uint16(data[body+2 : body+4]))
			format.SampleRate = int(binary.LittleEndian.Uint32(data[body+4 : body+8]))
			format.BitsPerSample = int(binary.LittleEndian.Uint16(data[body+14 : body+16]))
			format.Float = audioFormat == wavFormatFloat
			haveFmt = true
		case "data":
			sampleBytes = data[body : body+size]
		}

		pos = body + size
		if size%2 == 1 {
			pos++ // chunks are word-aligned
		}
	}

	if !haveFmt || sampleBytes == nil {
		return WAVFormat{}, nil, fmt.Errorf("tci: %s: missing fmt or data chunk", path)
	}
	if audioFormat != wavFormatPCM && audioFormat != wavFormatFloat {
		return WAVFormat{}, nil, fmt.Errorf("tci: %s: unsupported WAV audioFormat %d", path, audioFormat)
	}

	samples, err := pcmBytesToFloat32(sampleBytes, format)
	if err != nil {
		return WAVFormat{}, nil, fmt.Errorf("tci: %s: %w", path, err)
	}
	return format, samples, nil
}

func pcmBytesToFloat32(b []byte, f WAVFormat) ([]float32, error) {
	switch {
	case f.Float && f.BitsPerSample == 32:
		n := len(b) / 4
		out := make([]float32, n)
		for i := 0; i < n; i++ {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
		}
		return out, nil
	case !f.Float && f.BitsPerSample == 16:
		n := len(b) / 2
		out := make([]float32, n)
		for i := 0; i < n; i++ {
			out[i] = float32(int16(binary.LittleEndian.Uint16(b[i*2:]))) / 32768.0
		}
		return out, nil
	case !f.Float && f.BitsPerSample == 24:
		n := len(b) / 3
		out := make([]float32, n)
		for i := 0; i < n; i++ {
			v := int32(b[i*3]) | int32(b[i*3+1])<<8 | int32(b[i*3+2])<<16
			if v&0x800000 != 0 {
				v |= ^int32(0xFFFFFF)
			}
			out[i] = float32(v) / 8388608.0
		}
		return out, nil
	case !f.Float && f.BitsPerSample == 32:
		n := len(b) / 4
		out := make([]float32, n)
		for i := 0; i < n; i++ {
			out[i] = float32(int32(binary.LittleEndian.Uint32(b[i*4:]))) / 2147483648.0
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported bit depth %d (float=%v)", f.BitsPerSample, f.Float)
	}
}

// WriteWAV writes interleaved float32 samples in [-1, 1] as a canonical
// little-endian RIFF/WAVE file in the given format.
func WriteWAV(path string, f WAVFormat, samples []float32) error {
	if f.Channels < 1 {
		return fmt.Errorf("tci: WriteWAV: channels must be >= 1")
	}
	sampleBytes, err := float32ToPCMBytes(samples, f)
	if err != nil {
		return fmt.Errorf("tci: WriteWAV %s: %w", path, err)
	}

	blockAlign := f.Channels * (f.BitsPerSample / 8)
	byteRate := f.SampleRate * blockAlign
	dataSize := len(sampleBytes)

	buf := make([]byte, 0, 44+dataSize)
	buf = append(buf, "RIFF"...)
	buf = appendU32(buf, uint32(36+dataSize))
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	buf = appendU32(buf, 16)
	audioFormat := uint16(wavFormatPCM)
	if f.Float {
		audioFormat = wavFormatFloat
	}
	buf = appendU16(buf, audioFormat)
	buf = appendU16(buf, uint16(f.Channels))
	buf = appendU32(buf, uint32(f.SampleRate))
	buf = appendU32(buf, uint32(byteRate))
	buf = appendU16(buf, uint16(blockAlign))
	buf = appendU16(buf, uint16(f.BitsPerSample))
	buf = append(buf, "data"...)
	buf = appendU32(buf, uint32(dataSize))
	buf = append(buf, sampleBytes...)

	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("tci: write %s: %w", path, err)
	}
	return nil
}

func float32ToPCMBytes(samples []float32, f WAVFormat) ([]byte, error) {
	switch {
	case f.Float && f.BitsPerSample == 32:
		buf := make([]byte, len(samples)*4)
		for i, s := range samples {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(s))
		}
		return buf, nil
	case !f.Float && f.BitsPerSample == 16:
		buf := make([]byte, len(samples)*2)
		for i, s := range samples {
			v := int16(math.Round(float64(clip1(s)) * 32767))
			binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
		}
		return buf, nil
	case !f.Float && f.BitsPerSample == 24:
		buf := make([]byte, len(samples)*3)
		for i, s := range samples {
			v := int32(math.Round(float64(clip1(s)) * 8388607))
			buf[i*3] = byte(v)
			buf[i*3+1] = byte(v >> 8)
			buf[i*3+2] = byte(v >> 16)
		}
		return buf, nil
	case !f.Float && f.BitsPerSample == 32:
		buf := make([]byte, len(samples)*4)
		for i, s := range samples {
			v := int32(math.Round(float64(clip1(s)) * 2147483647))
			binary.LittleEndian.PutUint32(buf[i*4:], uint32(v))
		}
		return buf, nil
	default:
		return nil, fmt.Errorf("unsupported bit depth %d (float=%v)", f.BitsPerSample, f.Float)
	}
}

func clip1(s float32) float32 {
	if s > 1 {
		return 1
	}
	if s < -1 {
		return -1
	}
	return s
}

func appendU32(b []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendU16(b []byte, v uint16) []byte {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], v)
	return append(b, tmp[:]...)
}
