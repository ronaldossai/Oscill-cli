package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ronaldossai/Oscill-cli/internal/midi"
)

// midiLogCapacity caps how many recent MIDI events are retained in memory;
// the panel only ever displays as many as fit its height anyway.
const midiLogCapacity = 30

// midiTickInterval drives the telemetry panel's "time ago" column so it
// keeps advancing between events, not just when a new one arrives.
const midiTickInterval = 250 * time.Millisecond

var (
	midiLiveStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0A8A3E", Dark: "#7CE38B"})
	midiLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#CCCCCC"})
	midiUnmappedStyle = helpStyle
)

// midiLogEntry is one received MIDI note event, kept for the telemetry
// panel regardless of whether it matched a learned pad. Its pad mapping is
// resolved at render time (not stored here) since it can change after the
// fact — most notably, the tap that registers a brand new pad during
// learning wouldn't be "mapped" yet at the instant it's logged, but should
// still show as mapped once that pad exists.
type midiLogEntry struct {
	At       time.Time
	Note     uint8
	Velocity uint8
	On       bool
}

type midiTickMsg struct{}

func midiTickCmd() tea.Cmd {
	return tea.Tick(midiTickInterval, func(time.Time) tea.Msg { return midiTickMsg{} })
}

// logMIDIEvent records ev for the telemetry panel, newest first.
func (m *Model) logMIDIEvent(ev midi.NoteEvent) {
	entry := midiLogEntry{
		At:       time.Now(),
		Note:     ev.Key,
		Velocity: ev.Velocity,
		On:       ev.On,
	}
	m.midiLog = append([]midiLogEntry{entry}, m.midiLog...)
	if len(m.midiLog) > midiLogCapacity {
		m.midiLog = m.midiLog[:midiLogCapacity]
	}
}

// midiPanelView renders the MIDI telemetry panel's content, padded to
// height lines and every line truncated to width so it can never wrap —
// a wrapped line would make this panel taller than the one it's rendered
// beside, misaligning their borders.
func (m Model) midiPanelView(height, width int) string {
	var lines []string

	if m.midiConnected {
		lines = append(lines, midiLiveStyle.Render(truncateLabel("● "+m.midiPortName, width)))
	} else {
		lines = append(lines, helpStyle.Render(truncateLabel("○ not connected (press m)", width)))
	}

	switch {
	case len(m.midiLog) == 0 && m.midiConnected:
		lines = append(lines, helpStyle.Render(truncateLabel("waiting for events…", width)))
	case len(m.midiLog) > 0:
		now := time.Now()
		for _, e := range m.midiLog {
			if len(lines) >= height {
				break
			}
			lines = append(lines, formatMIDILogLine(e, now, m.findPad(e.Note), width))
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

func formatMIDILogLine(e midiLogEntry, now time.Time, padIndex, width int) string {
	state := "ON "
	if !e.On {
		state = "OFF"
	}

	if padIndex < 0 {
		line := truncateLabel(fmt.Sprintf("%6s %s %3d v%-3d unmapped", ageString(now.Sub(e.At)), state, e.Note, e.Velocity), width)
		return midiUnmappedStyle.Render(line)
	}
	line := truncateLabel(fmt.Sprintf("%6s %s %3d v%-3d pad%d", ageString(now.Sub(e.At)), state, e.Note, e.Velocity, padIndex+1), width)
	return midiLogStyle.Render(line)
}

// ageString renders a duration as a short "time ago" label.
func ageString(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}
