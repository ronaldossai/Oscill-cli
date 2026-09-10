package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakePlayer is a samplePlayer test double that never touches real audio
// hardware, so playback logic can be exercised deterministically.
type fakePlayer struct {
	playCount  int
	stopCount  int
	lastPath   string
	lastFormat string
	failNext   error
	done       chan struct{}
}

func (f *fakePlayer) Play(path, sampleFormat string) (<-chan struct{}, error) {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	f.playCount++
	f.lastPath = path
	f.lastFormat = sampleFormat
	f.done = make(chan struct{})
	return f.done, nil
}

func (f *fakePlayer) Stop() {
	f.stopCount++
}

func newTestBrowser(t *testing.T) (Model, *fakePlayer) {
	t.Helper()
	root := buildTestLibrary(t)

	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	fp := &fakePlayer{}
	m.player = fp

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	m = sendKey(t, m, "tab") // focus the Samples pane so a sample is selected
	return m, fp
}

func TestSpaceStartsPlayback(t *testing.T) {
	m, fp := newTestBrowser(t)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)

	if !m.playing {
		t.Fatal("expected m.playing = true after pressing space")
	}
	if fp.playCount != 1 {
		t.Fatalf("Play called %d times, want 1", fp.playCount)
	}
	if fp.lastFormat != "wav" {
		t.Errorf("lastFormat = %q, want %q", fp.lastFormat, "wav")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil command to wait for playback completion")
	}

	view := m.View()
	if !strings.Contains(view, "Playing") {
		t.Errorf("expected a playing indicator in the view, got:\n%s", view)
	}
}

func TestSpaceTwiceStopsPlayback(t *testing.T) {
	m, fp := newTestBrowser(t)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)

	if m.playing {
		t.Fatal("expected m.playing = false after pressing space twice")
	}
	if fp.stopCount != 1 {
		t.Fatalf("Stop called %d times, want 1", fp.stopCount)
	}
	if cmd != nil {
		t.Fatal("expected no wait-for-playback command when stopping")
	}
}

func TestPlaybackCompletionClearsPlayingState(t *testing.T) {
	m, fp := newTestBrowser(t)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("expected a wait-for-playback command")
	}

	close(fp.done) // simulate the sample finishing naturally

	msg := cmd() // run the command synchronously; it returns as soon as done is closed
	next, _ = m.Update(msg)
	m = next.(Model)

	if m.playing {
		t.Error("expected m.playing = false once playback finishes")
	}
}

func TestStalePlaybackCompletionIsIgnored(t *testing.T) {
	m, fp := newTestBrowser(t)

	// Descend into Kicks, which has two samples, so there's a second one to
	// switch to.
	m = sendKey(t, m, "tab") // back to the Folders pane
	m = sendKey(t, m, "enter")
	m = sendKey(t, m, "tab") // to the Samples pane

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)
	staleDone := fp.done

	// Move to the next sample and start playing it before the first one's
	// completion message arrives.
	m = sendKey(t, m, "down")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)
	if fp.playCount != 2 {
		t.Fatalf("Play called %d times, want 2", fp.playCount)
	}

	close(staleDone)
	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(Model)

	if !m.playing {
		t.Error("a stale completion message should not clear the current playback state")
	}
}

func TestPlayErrorSetsStatus(t *testing.T) {
	m, fp := newTestBrowser(t)
	fp.failNext = errors.New("boom")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(Model)

	if m.playing {
		t.Error("expected m.playing = false after a failed Play call")
	}
	if m.status == "" {
		t.Error("expected an error status after a failed Play call")
	}
	if cmd != nil {
		t.Error("expected no wait-for-playback command after a failed Play call")
	}
}
