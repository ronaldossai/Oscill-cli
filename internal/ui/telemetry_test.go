package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ronaldossai/Oscill-cli/internal/midi"
)

func TestLogMIDIEventCapsAtCapacity(t *testing.T) {
	var m Model
	for i := 0; i < midiLogCapacity+10; i++ {
		m.logMIDIEvent(midi.NoteEvent{Key: uint8(i % 128), Velocity: 100, On: true})
	}

	if len(m.midiLog) != midiLogCapacity {
		t.Fatalf("len(midiLog) = %d, want %d", len(m.midiLog), midiLogCapacity)
	}
	// Newest first: the very last event logged should be at index 0.
	if got, want := m.midiLog[0].Note, uint8((midiLogCapacity+9)%128); got != want {
		t.Errorf("midiLog[0].Note = %d, want %d (most recent)", got, want)
	}
}

func TestMIDIPanelViewPadMappingReflectsCurrentPads(t *testing.T) {
	var m Model
	m.midiConnected = true
	m.midiPortName = "Test Port"
	m.logMIDIEvent(midi.NoteEvent{Key: 36, Velocity: 100, On: true})

	// Not yet a known pad: should render as unmapped.
	view := m.midiPanelView(10, 60)
	if !strings.Contains(view, "unmapped") {
		t.Errorf("expected unmapped before the pad exists, got:\n%s", view)
	}

	// Once note 36 is a learned pad, the same log entry should now show it.
	m.pads = []padSlot{{Note: 36}}
	view = m.midiPanelView(10, 60)
	if !strings.Contains(view, "pad1") {
		t.Errorf("expected pad1 once note 36 is a learned pad, got:\n%s", view)
	}
	if strings.Contains(view, "unmapped") {
		t.Errorf("did not expect \"unmapped\" once note 36 is a learned pad, got:\n%s", view)
	}
}

func TestMIDIPanelViewNeverWrapsOrOverflowsHeight(t *testing.T) {
	var m Model
	m.midiConnected = true
	m.midiPortName = "A MIDI Controller With An Unreasonably Long Name"
	for i := 0; i < 5; i++ {
		m.logMIDIEvent(midi.NoteEvent{Key: uint8(36 + i), Velocity: 100, On: true})
	}
	m.pads = []padSlot{{Note: 36}, {Note: 37}}

	for _, width := range []int{10, 18, 30, 60} {
		height := 6
		view := m.midiPanelView(height, width)
		lines := strings.Split(view, "\n")
		if len(lines) != height {
			t.Errorf("width=%d: got %d lines, want exactly %d", width, len(lines), height)
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width=%d: line %d (%q) renders %d cells wide, want <= %d", width, i, line, w, width)
			}
		}
	}
}

func TestAgeString(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ms"},
		{500 * time.Millisecond, "500ms"},
		{999 * time.Millisecond, "999ms"},
		{1500 * time.Millisecond, "1.5s"},
		{59 * time.Second, "59.0s"},
		{90 * time.Second, "1m"},
		{-time.Second, "0ms"}, // clamp negative (clock skew) to zero
	}
	for _, tt := range tests {
		if got := ageString(tt.d); got != tt.want {
			t.Errorf("ageString(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
