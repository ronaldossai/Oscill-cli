package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDiscoversSupportedFormatsRecursively(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(rel string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	mustWrite("kick.wav")
	mustWrite("Kicks/kick_808.WAV") // extension case should not matter
	mustWrite("loop.flac")
	mustWrite("readme.txt") // should be ignored
	mustWrite(".DS_Store")  // should be ignored

	samples, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if len(samples) != 3 {
		names := make([]string, len(samples))
		for i, s := range samples {
			names[i] = s.Path
		}
		t.Fatalf("Scan found %d samples, want 3: %v", len(samples), names)
	}

	for _, s := range samples {
		if s.Size != 4 {
			t.Errorf("%s: Size = %d, want 4", s.Name, s.Size)
		}
		if s.HasAudioMetadata() && s.Format != FormatWAV {
			t.Errorf("%s: unexpected audio metadata for non-WAV placeholder file", s.Name)
		}
	}
}

func TestScanEmptyDirectory(t *testing.T) {
	samples, err := Scan(t.TempDir())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("Scan found %d samples in empty dir, want 0", len(samples))
	}
}
