package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestWAV writes a minimal, valid PCM WAV file for testing.
func writeTestWAV(t *testing.T, path string, sampleRate uint32, channels, bitsPerSample uint16, frames int) {
	t.Helper()

	blockAlign := channels * (bitsPerSample / 8)
	dataSize := uint32(frames) * uint32(blockAlign)
	byteRate := sampleRate * uint32(blockAlign)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating test wav: %v", err)
	}
	defer f.Close()

	write := func(v any) {
		if err := binary.Write(f, binary.LittleEndian, v); err != nil {
			t.Fatalf("writing wav field: %v", err)
		}
	}

	f.WriteString("RIFF")
	write(uint32(36 + dataSize))
	f.WriteString("WAVE")

	f.WriteString("fmt ")
	write(uint32(16)) // fmt chunk size
	write(uint16(1))  // PCM
	write(channels)
	write(sampleRate)
	write(byteRate)
	write(blockAlign)
	write(bitsPerSample)

	f.WriteString("data")
	write(dataSize)
	f.Write(make([]byte, dataSize))
}

func TestProbeWAV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")
	writeTestWAV(t, path, 44100, 2, 16, 44100) // 1 second, stereo, 16-bit

	meta, err := Probe(path, "wav")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}

	if meta.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", meta.SampleRate)
	}
	if meta.Channels != 2 {
		t.Errorf("Channels = %d, want 2", meta.Channels)
	}
	if meta.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", meta.BitDepth)
	}
	if diff := meta.Duration - time.Second; diff < -time.Millisecond || diff > time.Millisecond {
		t.Errorf("Duration = %v, want ~1s", meta.Duration)
	}
}

func TestProbeUnsupportedFormat(t *testing.T) {
	_, err := Probe("irrelevant.flac", "flac")
	if err != ErrUnsupportedFormat {
		t.Errorf("Probe() error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestProbeWAVRejectsNonWAV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-wav.wav")
	if err := os.WriteFile(path, []byte("not a real wav file"), 0o644); err != nil {
		t.Fatalf("writing bogus file: %v", err)
	}

	if _, err := Probe(path, "wav"); err == nil {
		t.Error("Probe() error = nil, want error for invalid WAV header")
	}
}
