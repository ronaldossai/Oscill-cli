// Package ui implements Oscill's interactive terminal sample browser.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ronaldossai/Oscill-cli/internal/format"
	"github.com/ronaldossai/Oscill-cli/internal/library"
)

type focusedPane int

const (
	focusFolders focusedPane = iota
	focusFiles
)

const (
	headerLines        = 1
	sectionHeaderLines = 1
	blankLines         = 1
	blankGaps          = 4 // header→folders, folders→files, files→metadata, metadata→footer
	metadataBoxLines   = 5
	footerLines        = 1
	minPaneHeight      = 3
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0060D0", Dark: "#7AD1FF"})
	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"})
	sectionStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#CCCCCC"})
	focusedSectionStyle = sectionStyle.
				Foreground(lipgloss.AdaptiveColor{Light: "#0060D0", Dark: "#7AD1FF"})
	metadataBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#555555"}).
				Padding(0, 1)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#B00020", Dark: "#FF6B6B"})
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})
)

// Model is the Bubble Tea model for the interactive sample browser.
type Model struct {
	root string
	dir  string

	folders list.Model
	files   list.Model
	focus   focusedPane

	width, height int
	status        string
}

// NewBrowser creates a browser rooted at root. Navigation with "up a
// directory" will not go above root.
func NewBrowser(root string) (Model, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Model{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Model{}, err
	}
	if !info.IsDir() {
		return Model{}, fmt.Errorf("%s is not a directory", abs)
	}

	m := Model{root: abs, dir: abs, focus: focusFolders}

	m.folders = list.New(nil, itemDelegate{}, 0, 0)
	m.files = list.New(nil, itemDelegate{}, 0, 0)
	for _, l := range []*list.Model{&m.folders, &m.files} {
		l.SetShowTitle(false)
		l.SetShowStatusBar(false)
		l.SetShowHelp(false)
		l.SetShowPagination(false)
		l.SetFilteringEnabled(false)
		l.DisableQuitKeybindings()
	}

	if err := m.load(abs); err != nil {
		return Model{}, err
	}
	return m, nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applySizes()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			if m.focus == focusFolders {
				m.focus = focusFiles
			} else {
				m.focus = focusFolders
			}
			return m, nil

		case "enter":
			if m.focus == focusFolders {
				if item, ok := m.folders.SelectedItem().(dirItem); ok {
					next := filepath.Join(m.dir, string(item))
					if err := m.load(next); err != nil {
						m.status = err.Error()
					}
				}
			}
			return m, nil

		case "backspace", "h", "left":
			if m.dir == m.root {
				m.status = "already at the library root"
			} else if err := m.load(filepath.Dir(m.dir)); err != nil {
				m.status = err.Error()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.focus == focusFolders {
		m.folders, cmd = m.folders.Update(msg)
	} else {
		m.files, cmd = m.files.Update(msg)
	}
	return m, cmd
}

// load reads dir's contents and replaces the folders/files panes with them.
func (m *Model) load(dir string) error {
	listing, err := library.ListDir(dir)
	if err != nil {
		return err
	}

	dirItems := make([]list.Item, len(listing.Dirs))
	for i, d := range listing.Dirs {
		dirItems[i] = dirItem(d)
	}
	sampleItems := make([]list.Item, len(listing.Samples))
	for i, s := range listing.Samples {
		sampleItems[i] = sampleItem(s)
	}

	m.dir = dir
	m.status = ""
	m.folders.SetItems(dirItems)
	m.files.SetItems(sampleItems)
	m.folders.Select(0)
	m.files.Select(0)
	return nil
}

func (m *Model) applySizes() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	chrome := headerLines + sectionHeaderLines*2 + blankLines*blankGaps + metadataBoxLines + footerLines
	available := m.height - chrome
	if available < minPaneHeight*2 {
		available = minPaneHeight * 2
	}

	foldersHeight := available * 2 / 5
	if foldersHeight < minPaneHeight {
		foldersHeight = minPaneHeight
	}
	filesHeight := available - foldersHeight
	if filesHeight < minPaneHeight {
		filesHeight = minPaneHeight
	}

	m.folders.SetSize(m.width, foldersHeight)
	m.files.SetSize(m.width, filesHeight)
}

const (
	minTerminalWidth  = 40
	minTerminalHeight = 20
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	if m.width < minTerminalWidth || m.height < minTerminalHeight {
		return fmt.Sprintf("Terminal too small (need at least %dx%d).", minTerminalWidth, minTerminalHeight)
	}

	rel, err := filepath.Rel(m.root, m.dir)
	if err != nil || rel == "." {
		rel = "/"
	} else {
		rel = "/" + rel
	}

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("OSCILL")+"  "+pathStyle.Render(rel))
	fmt.Fprintln(&b)

	foldersHeader := "Folders"
	filesHeader := "Samples"
	if m.focus == focusFolders {
		foldersHeader = focusedSectionStyle.Render(foldersHeader)
	} else {
		foldersHeader = sectionStyle.Render(foldersHeader)
	}
	if m.focus == focusFiles {
		filesHeader = focusedSectionStyle.Render(filesHeader)
	} else {
		filesHeader = sectionStyle.Render(filesHeader)
	}

	fmt.Fprintln(&b, foldersHeader)
	fmt.Fprintln(&b, m.folders.View())
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, filesHeader)
	fmt.Fprintln(&b, m.files.View())
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, metadataBoxStyle.Width(m.width-4).Render(m.metadataView()))
	fmt.Fprintln(&b)

	help := "↑/↓ or j/k move   tab switch pane   enter open folder   backspace up   q quit"
	if m.status != "" {
		fmt.Fprint(&b, statusStyle.Render(m.status))
	} else {
		fmt.Fprint(&b, helpStyle.Render(help))
	}

	return b.String()
}

// metadataView always returns exactly 3 lines, so the metadata box's
// rendered height stays constant regardless of selection state (see
// metadataBoxLines).
func (m Model) metadataView() string {
	item := m.files.SelectedItem()
	if item == nil {
		return "No sample selected.\n\n"
	}
	s := library.Sample(item.(sampleItem))

	return fmt.Sprintf(
		"%s\nFormat: %s   Size: %s   Duration: %s\nSample Rate: %s   Channels: %s   Bit Depth: %s",
		s.Name,
		s.Format,
		format.HumanSize(s.Size),
		format.OptionalDuration(s.Duration, s.HasAudioMetadata()),
		format.OptionalInt(s.SampleRate, s.HasAudioMetadata(), "Hz"),
		format.OptionalInt(s.Channels, s.HasAudioMetadata(), ""),
		format.OptionalInt(s.BitDepth, s.HasAudioMetadata(), "-bit"),
	)
}
