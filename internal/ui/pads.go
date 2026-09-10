package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ronaldossai/Oscill-cli/internal/library"
	"github.com/ronaldossai/Oscill-cli/internal/midi"
)

// learnIdleTimeout is how long the pad grid waits after the last pad tap
// before deciding no more pads are coming.
const learnIdleTimeout = 2 * time.Second

// flashDuration is how long a pad cell stays highlighted after being hit.
const flashDuration = 150 * time.Millisecond

var (
	padCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#CCCCCC"})
	padFlashStyle = lipgloss.NewStyle().Bold(true).Reverse(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0060D0", Dark: "#7AD1FF"})
)

// padSlot is one learned MIDI pad, identified by the note number it sends,
// optionally bound to a sample to play when hit.
type padSlot struct {
	Note   uint8
	Sample *library.Sample
}

// midiNoteMsg wraps a note event delivered from the connected MIDI input.
type midiNoteMsg midi.NoteEvent

// midiClosedMsg reports that the MIDI input channel was closed (the
// connectMIDI-provided channel will not deliver anything further).
type midiClosedMsg struct{}

// learnIdleMsg reports that no new pad has been tapped for learnIdleTimeout.
// It carries the learning "generation" active when the timer was armed, so a
// timer superseded by a later tap (which arms its own, newer timer) is
// recognized as stale and ignored.
type learnIdleMsg struct{ gen int }

// padFlashDoneMsg ends a pad's brief highlight after being hit. It carries
// the flash "generation" for the same stale-message reason as learnIdleMsg.
type padFlashDoneMsg struct{ gen int }

// waitForMIDI returns a command that blocks for the next event on events,
// then reports it back to Update. It runs on Bubble Tea's own goroutine, so
// it does not block the UI, and it must be re-issued after every event to
// keep listening (see handleMIDINote).
func waitForMIDI(events <-chan midi.NoteEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return midiClosedMsg{}
		}
		return midiNoteMsg(ev)
	}
}

func learnIdleCmd(gen int) tea.Cmd {
	return tea.Tick(learnIdleTimeout, func(time.Time) tea.Msg {
		return learnIdleMsg{gen: gen}
	})
}

// connectOrRelearnMIDI opens the first available MIDI input on first use
// (subsequent presses just restart pad learning on the already-open port).
func (m Model) connectOrRelearnMIDI() (tea.Model, tea.Cmd) {
	if !m.midiConnected {
		ports, err := m.listMIDIPorts()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		if len(ports) == 0 {
			m.status = "no MIDI input device found"
			return m, nil
		}

		events, closeFn, err := m.connectMIDI(ports[0])
		if err != nil {
			m.status = err.Error()
			return m, nil
		}

		m.midiConnected = true
		m.midiPortName = ports[0]
		m.midiClose = closeFn
		m.midiEvents = events
		m.pads = nil
		m.learning = true
		m.learnGen++
		m.status = fmt.Sprintf("Connected to %s — tap each pad...", ports[0])
		return m, tea.Batch(waitForMIDI(events), learnIdleCmd(m.learnGen))
	}

	m.pads = nil
	m.learning = true
	m.assigning = false
	m.learnGen++
	m.status = "Tap each pad again..."
	return m, learnIdleCmd(m.learnGen)
}

// armAssign puts the browser into "tap a pad to assign" mode for the
// currently selected sample.
func (m Model) armAssign() (tea.Model, tea.Cmd) {
	if !m.midiConnected || len(m.pads) == 0 {
		m.status = "connect a MIDI controller first (press m)"
		return m, nil
	}
	item, ok := m.files.SelectedItem().(sampleItem)
	if !ok {
		m.status = "select a sample to assign"
		return m, nil
	}

	m.assigning = true
	m.assignSample = item.Sample
	m.status = fmt.Sprintf("Tap a pad to assign %s… (esc to cancel)", m.assignSample.Name)
	return m, nil
}

func (m Model) findPad(note uint8) int {
	for i, p := range m.pads {
		if p.Note == note {
			return i
		}
	}
	return -1
}

// handleMIDINote processes one note event and always re-arms MIDI listening
// for the next one.
func (m Model) handleMIDINote(msg midiNoteMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{waitForMIDI(m.midiEvents)}

	if !msg.On {
		return m, tea.Batch(cmds...)
	}

	if m.learning {
		if m.findPad(msg.Key) < 0 {
			m.pads = append(m.pads, padSlot{Note: msg.Key})
		}
		m.learnGen++
		m.status = fmt.Sprintf("Learning pads… %d detected so far (pause to finish)", len(m.pads))
		cmds = append(cmds, learnIdleCmd(m.learnGen))
		return m, tea.Batch(cmds...)
	}

	idx := m.findPad(msg.Key)
	if idx < 0 {
		return m, tea.Batch(cmds...) // not a learned pad (e.g. the keybed) — ignore
	}

	if m.assigning {
		s := m.assignSample
		m.pads[idx].Sample = &s
		m.assigning = false
		m.status = fmt.Sprintf("Assigned %s to pad %d", s.Name, idx+1)
		return m, tea.Batch(cmds...)
	}

	m.flashGen++
	gen := m.flashGen
	m.flashPad = idx
	cmds = append(cmds, tea.Tick(flashDuration, func(time.Time) tea.Msg { return padFlashDoneMsg{gen: gen} }))

	if pad := m.pads[idx]; pad.Sample != nil {
		done, err := m.player.Play(pad.Sample.Path, string(pad.Sample.Format))
		if err != nil {
			m.status = err.Error()
		} else {
			m.playing = true
			m.playingPath = pad.Sample.Path
			m.playingDone = done
			m.status = ""
			cmds = append(cmds, waitForPlayback(done))
		}
	} else {
		m.status = fmt.Sprintf("Pad %d is empty — press 'a' on a sample to assign it", idx+1)
	}

	return m, tea.Batch(cmds...)
}

// padsView renders the pad grid's single content line.
func (m Model) padsView() string {
	if !m.midiConnected {
		return "Press 'm' to connect a MIDI controller."
	}
	if m.learning {
		return fmt.Sprintf("Learning pads on %s… %d detected (tap more, pause to finish)", m.midiPortName, len(m.pads))
	}
	if len(m.pads) == 0 {
		return fmt.Sprintf("Connected to %s — no pads learned (press 'm' to learn)", m.midiPortName)
	}

	cells := make([]string, len(m.pads))
	for i, pad := range m.pads {
		label := "--"
		if pad.Sample != nil {
			label = truncateLabel(pad.Sample.Name, 8)
		}
		cell := fmt.Sprintf("%d:%s", i+1, label)
		if i == m.flashPad {
			cells[i] = padFlashStyle.Render(cell)
		} else {
			cells[i] = padCellStyle.Render(cell)
		}
	}
	return strings.Join(cells, " ")
}

func truncateLabel(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
