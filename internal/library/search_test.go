package library

import (
	"testing"
	"time"
)

func TestQueryMatch(t *testing.T) {
	known := Sample{
		Path: "/lib/Kicks/kick_808.wav", Name: "kick_808.wav",
		Format: FormatWAV, Duration: 600 * time.Millisecond, SampleRate: 44100, Channels: 1, BitDepth: 16,
	}
	unknownDuration := Sample{
		Path: "/lib/loop.flac", Name: "loop.flac", Format: FormatFLAC,
	}

	tests := []struct {
		name  string
		q     Query
		s     Sample
		match bool
	}{
		{"empty query matches anything", Query{}, known, true},
		{"name matches filename", Query{Name: "kick"}, known, true},
		{"name matches folder", Query{Name: "Kicks"}, known, true},
		{"name is case-insensitive", Query{Name: "KICK"}, known, true},
		{"name mismatch", Query{Name: "snare"}, known, false},
		{"format match", Query{Format: FormatWAV}, known, true},
		{"format mismatch", Query{Format: FormatFLAC}, known, false},
		{"min duration satisfied", Query{MinDuration: 500 * time.Millisecond}, known, true},
		{"min duration violated", Query{MinDuration: time.Second}, known, false},
		{"max duration satisfied", Query{MaxDuration: time.Second}, known, true},
		{"max duration violated", Query{MaxDuration: 500 * time.Millisecond}, known, false},
		{"duration bound excludes unknown-duration sample", Query{MaxDuration: 2 * time.Second}, unknownDuration, false},
		{"name query still applies to unknown-duration sample", Query{Name: "loop"}, unknownDuration, true},
		{"combined constraints all satisfied", Query{Name: "kick", Format: FormatWAV, MaxDuration: time.Second}, known, true},
		{"combined constraints one violated", Query{Name: "kick", Format: FormatFLAC}, known, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.Match(tt.s); got != tt.match {
				t.Errorf("Match() = %v, want %v", got, tt.match)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	samples := []Sample{
		{Path: "/lib/Kicks/kick_808.wav", Name: "kick_808.wav", Format: FormatWAV},
		{Path: "/lib/Kicks/kick_deep.wav", Name: "kick_deep.wav", Format: FormatWAV},
		{Path: "/lib/Snares/snare_tight.wav", Name: "snare_tight.wav", Format: FormatWAV},
		{Path: "/lib/loop.flac", Name: "loop.flac", Format: FormatFLAC},
	}

	got := Filter(samples, Query{Name: "kick"})
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	for _, s := range got {
		if s.Format != FormatWAV {
			t.Errorf("unexpected result %s", s.Path)
		}
	}

	if got := Filter(samples, Query{}); len(got) != len(samples) {
		t.Errorf("empty query returned %d results, want all %d", len(got), len(samples))
	}

	if got := Filter(samples, Query{Name: "nonexistent"}); len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}
