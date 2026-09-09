package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// probeWAV reads a WAV file's RIFF chunk headers to recover its format and
// duration without decoding any sample data.
func probeWAV(path string) (Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return Metadata{}, err
	}
	defer f.Close()

	var riffHeader [12]byte
	if _, err := io.ReadFull(f, riffHeader[:]); err != nil {
		return Metadata{}, fmt.Errorf("audio: reading RIFF header: %w", err)
	}
	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return Metadata{}, fmt.Errorf("audio: %s is not a valid WAV file", path)
	}

	var (
		sampleRate    uint32
		channels      uint16
		bitsPerSample uint16
		dataSize      uint32
		haveFmt       bool
		haveData      bool
	)

	var chunkHeader [8]byte
	for {
		if _, err := io.ReadFull(f, chunkHeader[:]); err != nil {
			break // EOF (or truncated file) once we run out of chunks.
		}
		id := string(chunkHeader[0:4])
		size := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch id {
		case "fmt ":
			body := make([]byte, size)
			if _, err := io.ReadFull(f, body); err != nil {
				return Metadata{}, fmt.Errorf("audio: reading fmt chunk: %w", err)
			}
			if len(body) < 16 {
				return Metadata{}, fmt.Errorf("audio: %s has a malformed fmt chunk", path)
			}
			channels = binary.LittleEndian.Uint16(body[2:4])
			sampleRate = binary.LittleEndian.Uint32(body[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(body[14:16])
			haveFmt = true
		case "data":
			dataSize = size
			haveData = true
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return Metadata{}, fmt.Errorf("audio: skipping data chunk: %w", err)
			}
		default:
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return Metadata{}, fmt.Errorf("audio: skipping %q chunk: %w", id, err)
			}
		}

		if size%2 == 1 {
			// Chunks are word-aligned; skip the padding byte.
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				break
			}
		}

		if haveFmt && haveData {
			break
		}
	}

	if !haveFmt || !haveData {
		return Metadata{}, fmt.Errorf("audio: %s is missing required WAV chunks", path)
	}
	if sampleRate == 0 || channels == 0 || bitsPerSample == 0 {
		return Metadata{}, fmt.Errorf("audio: %s has an invalid fmt chunk", path)
	}

	bytesPerSecond := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	duration := time.Duration(0)
	if bytesPerSecond > 0 {
		duration = time.Duration(float64(dataSize) / float64(bytesPerSecond) * float64(time.Second))
	}

	return Metadata{
		Duration:   duration,
		SampleRate: int(sampleRate),
		Channels:   int(channels),
		BitDepth:   int(bitsPerSample),
	}, nil
}
