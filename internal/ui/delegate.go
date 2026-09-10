package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	normalItemStyle   = lipgloss.NewStyle()
	selectedItemStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#0060D0", Dark: "#7AD1FF"})
)

// itemDelegate renders a single-line entry per item (no title/description
// split like the bubbles default delegate), matching a compact file-browser
// look.
type itemDelegate struct{}

func (d itemDelegate) Height() int                         { return 1 }
func (d itemDelegate) Spacing() int                        { return 0 }
func (d itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var name string
	switch it := item.(type) {
	case dirItem:
		name = string(it) + "/"
	case sampleItem:
		name = it.Display
	default:
		return
	}

	prefix := "  "
	style := normalItemStyle
	if index == m.Index() {
		prefix = "> "
		style = selectedItemStyle
	}
	fmt.Fprint(w, style.Render(prefix+name))
}
