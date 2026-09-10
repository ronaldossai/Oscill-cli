package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ronaldossai/Oscill-cli/internal/midi"
)

// newTestBrowserWithMIDI wires a fake single MIDI port backed by a channel
// the test controls directly, so pad learning/assignment/playback can be
// exercised without any real MIDI hardware.
func newTestBrowserWithMIDI(t *testing.T) (Model, *fakePlayer, chan midi.NoteEvent) {
	t.Helper()
	m, fp := newTestBrowser(t)

	events := make(chan midi.NoteEvent, 8)
	m.listMIDIPorts = func() ([]string, error) { return []string{"Fake MIDI Port"}, nil }
	m.connectMIDI = func(port string) (<-chan midi.NoteEvent, func(), error) {
		return events, func() {}, nil
	}

	return m, fp, events
}

// pressNote pushes a note-on event for key onto events and feeds it through
// Update the same way the running program would once the corresponding
// waitForMIDI command fires. It sidesteps whatever exact tea.Cmd Update
// returned (often a tea.Batch mixing in a timer) since that's awkward to
// unwrap in a test; m.midiEvents is always the right channel to wait on
// regardless of how the previous command was batched.
func pressNote(t *testing.T, m Model, events chan midi.NoteEvent, key uint8) Model {
	t.Helper()
	events <- midi.NoteEvent{Key: key, Velocity: 100, On: true}
	next, _ := m.Update(waitForMIDI(m.midiEvents)())
	return next.(Model)
}

func connectFakeMIDI(t *testing.T, m Model) Model {
	t.Helper()
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m = next.(Model)
	if !m.midiConnected {
		t.Fatal("expected midiConnected = true after pressing m")
	}
	if cmd == nil {
		t.Fatal("expected a command after connecting")
	}
	return m
}

func TestMIDIConnectAndLearnPads(t *testing.T) {
	m, _, events := newTestBrowserWithMIDI(t)
	m = connectFakeMIDI(t, m)

	if !m.learning {
		t.Fatal("expected learning = true right after connecting")
	}

	for _, key := range []uint8{36, 37, 38} {
		m = pressNote(t, m, events, key)
	}

	if len(m.pads) != 3 {
		t.Fatalf("len(m.pads) = %d, want 3", len(m.pads))
	}
	if !m.learning {
		t.Fatal("expected still learning before the idle timeout fires")
	}

	// Re-tapping an already-learned pad should not add a duplicate.
	m = pressNote(t, m, events, 36)
	if len(m.pads) != 3 {
		t.Fatalf("len(m.pads) = %d after re-tapping pad 1, want 3", len(m.pads))
	}

	// A stale idle timeout (from before the re-taps extended it) must not
	// end learning early.
	next, _ := m.Update(learnIdleMsg{gen: m.learnGen - 1})
	m = next.(Model)
	if !m.learning {
		t.Fatal("a stale learnIdleMsg incorrectly ended learning")
	}

	// The current generation's timeout finalizes it.
	next, _ = m.Update(learnIdleMsg{gen: m.learnGen})
	m = next.(Model)
	if m.learning {
		t.Fatal("expected learning = false after the current idle timeout fires")
	}
	if len(m.pads) != 3 {
		t.Fatalf("len(m.pads) = %d after learning finished, want 3", len(m.pads))
	}
}

func TestAssignAndTriggerPad(t *testing.T) {
	m, fp, events := newTestBrowserWithMIDI(t)
	m = connectFakeMIDI(t, m)
	m = pressNote(t, m, events, 36)
	next, _ := m.Update(learnIdleMsg{gen: m.learnGen})
	m = next.(Model)

	if len(m.pads) != 1 || m.learning {
		t.Fatalf("setup failed: pads=%d learning=%v", len(m.pads), m.learning)
	}

	// newTestBrowser already focused the Samples pane and selected its first
	// sample; arm assignment for it.
	next, acmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = next.(Model)
	if acmd != nil {
		t.Fatal("armAssign should not return a command")
	}
	if !m.assigning {
		t.Fatal("expected assigning = true after pressing 'a'")
	}

	// Tap the learned pad to complete the assignment.
	m = pressNote(t, m, events, 36)
	if m.assigning {
		t.Fatal("expected assigning = false after tapping the pad")
	}
	if m.pads[0].Sample == nil {
		t.Fatal("expected pad 0 to have a sample assigned")
	}

	// Hitting the pad again should now play the assigned sample and flash it.
	m = pressNote(t, m, events, 36)
	if fp.playCount != 1 {
		t.Fatalf("Play called %d times, want 1", fp.playCount)
	}
	if m.flashPad != 0 {
		t.Fatalf("flashPad = %d, want 0", m.flashPad)
	}

	view := m.padsView()
	if !strings.Contains(view, "1:") {
		t.Errorf("expected pad 1 in pads view, got: %s", view)
	}
}

func TestUnknownNoteIsIgnoredWhenNotLearning(t *testing.T) {
	m, fp, events := newTestBrowserWithMIDI(t)
	m = connectFakeMIDI(t, m)
	m = pressNote(t, m, events, 36)
	next, _ := m.Update(learnIdleMsg{gen: m.learnGen})
	m = next.(Model)

	// A note that was never learned (e.g. the keybed) should be ignored.
	m = pressNote(t, m, events, 90)

	if fp.playCount != 0 {
		t.Errorf("Play called %d times, want 0 for an unlearned note", fp.playCount)
	}
	if m.flashPad != -1 {
		t.Errorf("flashPad = %d, want -1 for an unlearned note", m.flashPad)
	}
}

func TestArmAssignRequiresConnectedPadsAndSelection(t *testing.T) {
	m, _ := newTestBrowser(t) // no MIDI wired up at all

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = next.(Model)
	if cmd != nil {
		t.Error("expected no command from armAssign without a MIDI connection")
	}
	if m.assigning {
		t.Error("expected assigning = false without a MIDI connection")
	}
	if m.status == "" {
		t.Error("expected a status message explaining why assignment was refused")
	}
}

func TestNoMIDIPortsSetsStatus(t *testing.T) {
	m, _ := newTestBrowser(t)
	m.listMIDIPorts = func() ([]string, error) { return nil, nil }
	m.connectMIDI = func(string) (<-chan midi.NoteEvent, func(), error) {
		return nil, nil, errors.New("should not be called")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m = next.(Model)

	if m.midiConnected {
		t.Error("expected midiConnected = false when no ports are available")
	}
	if m.status == "" {
		t.Error("expected a status message when no MIDI ports are found")
	}
}
