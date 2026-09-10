// Package ui implements Oscill's interactive terminal sample browser.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ronaldossai/Oscill-cli/internal/audio"
	"github.com/ronaldossai/Oscill-cli/internal/format"
	"github.com/ronaldossai/Oscill-cli/internal/library"
	"github.com/ronaldossai/Oscill-cli/internal/midi"
)

// samplePlayer is the subset of *audio.Player the browser depends on,
// factored out so tests can substitute a fake instead of touching real
// audio hardware.
type samplePlayer interface {
	Play(path string, sampleFormat string) (<-chan struct{}, error)
	Stop()
}

type focusedPane int

const (
	focusFolders focusedPane = iota
	focusFiles
)

// oscillBanner is the block-letter "OSCILL" wordmark shown at the top of
// the browser (generated with the figlet "small" font).
const oscillBanner = `   ___    ___    ___   ___   _      _
  / _ \  / __|  / __| |_ _| | |    | |
 | (_) | \__ \ | (__   | |  | |__  | |__
  \___/  |___/  \___| |___| |____| |____|`

const bannerLines = 4

const (
	headerLines      = bannerLines + 1 // banner + path line
	blankLines       = 1
	blankGaps        = 5 // header→folders, folders→files, files→pads, pads→metadata, metadata→footer
	panelChromeLines = 3 // title line + top/bottom border, per bordered panel
	padsBoxLines     = 4 // title line + 1 content line + top/bottom border
	metadataBoxLines = 5
	footerLines      = 1
	minPaneHeight    = 3
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
	panelBorderColor        = lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#555555"}
	focusedPanelBorderColor = lipgloss.AdaptiveColor{Light: "#0060D0", Dark: "#7AD1FF"}
	metadataBoxStyle        = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(panelBorderColor).
				Padding(0, 1)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#B00020", Dark: "#FF6B6B"})
	playingStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0A8A3E", Dark: "#7CE38B"})
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})
)

// panelStyle returns a bordered panel container, highlighted when it holds
// keyboard focus.
func panelStyle(focused bool) lipgloss.Style {
	color := panelBorderColor
	if focused {
		color = focusedPanelBorderColor
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1)
}

// Model is the Bubble Tea model for the interactive sample browser.
type Model struct {
	root string
	dir  string

	folders list.Model
	files   list.Model
	focus   focusedPane

	player      samplePlayer
	playing     bool
	playingPath string
	playingDone <-chan struct{}

	listMIDIPorts func() ([]string, error)
	connectMIDI   func(port string) (<-chan midi.NoteEvent, func(), error)

	midiConnected bool
	midiPortName  string
	midiClose     func()
	midiEvents    <-chan midi.NoteEvent
	midiLog       []midiLogEntry

	pads     []padSlot
	learning bool
	learnGen int

	assigning    bool
	assignSample library.Sample

	flashPad int
	flashGen int

	searching    bool
	searchTyping bool
	searchInput  textinput.Model
	searchAll    []library.Sample

	width, height int
	status        string
	statusSetAt   time.Time
}

// statusTimeout is how long a status/error message stays in the footer
// before it reverts to the keybinding help text, so a one-off message like
// "already at the library root" doesn't permanently bury the help.
const statusTimeout = 3 * time.Second

// statusCheckInterval drives the check for whether the current status has
// aged past statusTimeout. It runs for the life of the program (started
// once from Init), independent of any other feature being active.
const statusCheckInterval = 250 * time.Millisecond

// setStatus sets the footer's status/error text and timestamps it for
// statusTickMsg to expire later. Passing "" clears it immediately.
func (m *Model) setStatus(s string) {
	m.status = s
	m.statusSetAt = time.Now()
}

type statusTickMsg struct{}

func statusTickCmd() tea.Cmd {
	return tea.Tick(statusCheckInterval, func(time.Time) tea.Msg { return statusTickMsg{} })
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

	m := Model{
		root: abs, dir: abs, focus: focusFolders,
		player:        audio.NewPlayer(),
		listMIDIPorts: midi.Ports,
		connectMIDI:   midi.Listen,
		flashPad:      -1,
	}

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

func (m Model) Init() tea.Cmd { return statusTickCmd() }

// playbackFinishedMsg reports that a previously started Play call's done
// channel has closed. It carries that channel so a stale completion (from a
// sample that was superseded by a newer Play call) can be told apart from
// the one currently playing.
type playbackFinishedMsg struct{ done <-chan struct{} }

// waitForPlayback returns a command that blocks until done is closed, then
// reports it back to Update. It runs on Bubble Tea's own goroutine, so it
// does not block the UI.
func waitForPlayback(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done
		return playbackFinishedMsg{done: done}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applySizes()
		return m, nil

	case playbackFinishedMsg:
		if m.playing && msg.done == m.playingDone {
			m.playing = false
			m.playingPath = ""
			m.playingDone = nil
		}
		return m, nil

	case midiNoteMsg:
		return m.handleMIDINote(msg)

	case midiClosedMsg:
		m.midiConnected = false
		m.midiPortName = ""
		m.pads = nil
		m.learning = false
		m.assigning = false
		m.setStatus("MIDI input disconnected")
		return m, nil

	case statusTickMsg:
		// Active-mode reminders (mid-learn, mid-assign) stay up until that
		// mode ends rather than expiring while the user still needs them.
		if m.status != "" && !m.assigning && !m.learning && time.Since(m.statusSetAt) >= statusTimeout {
			m.status = ""
		}
		return m, statusTickCmd()

	case midiTickMsg:
		if !m.midiConnected {
			return m, nil // let the loop end once disconnected
		}
		return m, midiTickCmd()

	case learnIdleMsg:
		if m.learning && msg.gen == m.learnGen {
			m.learning = false
			m.setStatus(fmt.Sprintf("Detected %d pad(s) on %s", len(m.pads), m.midiPortName))
		}
		return m, nil

	case padFlashDoneMsg:
		if msg.gen == m.flashGen {
			m.flashPad = -1
		}
		return m, nil

	case tea.KeyMsg:
		if m.searchTyping {
			return m.handleSearchTypingKey(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.player.Stop()
			if m.midiClose != nil {
				m.midiClose()
			}
			return m, tea.Quit

		case " ":
			return m.togglePlay()

		case "s":
			if m.playing {
				m.player.Stop()
				m.playing = false
				m.playingPath = ""
				m.playingDone = nil
			}
			return m, nil

		case "m":
			return m.connectOrRelearnMIDI()

		case "a":
			return m.armAssign()

		case "/":
			return m.startSearch()

		case "esc":
			switch {
			case m.assigning:
				m.assigning = false
				m.setStatus("")
			case m.searching:
				m.clearSearch()
			}
			return m, nil

		case "tab":
			if m.searching {
				return m, nil
			}
			if m.focus == focusFolders {
				m.focus = focusFiles
			} else {
				m.focus = focusFolders
			}
			return m, nil

		case "enter":
			if m.searching {
				return m.jumpToSearchResult()
			}
			if m.focus == focusFolders {
				if item, ok := m.folders.SelectedItem().(dirItem); ok {
					next := filepath.Join(m.dir, string(item))
					if err := m.load(next); err != nil {
						m.setStatus(err.Error())
					}
				}
			}
			return m, nil

		case "backspace", "h", "left":
			if m.searching {
				m.clearSearch()
				return m, nil
			}
			if m.dir == m.root {
				m.setStatus("already at the library root")
			} else if err := m.load(filepath.Dir(m.dir)); err != nil {
				m.setStatus(err.Error())
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

// togglePlay starts playing the selected sample, or stops it if it's
// already the one playing.
func (m Model) togglePlay() (tea.Model, tea.Cmd) {
	item, ok := m.files.SelectedItem().(sampleItem)
	if !ok {
		return m, nil
	}
	s := item.Sample

	if m.playing && m.playingPath == s.Path {
		m.player.Stop()
		m.playing = false
		m.playingPath = ""
		m.playingDone = nil
		return m, nil
	}

	done, err := m.player.Play(s.Path, string(s.Format))
	if err != nil {
		m.setStatus(err.Error())
		m.playing = false
		m.playingPath = ""
		m.playingDone = nil
		return m, nil
	}

	m.playing = true
	m.playingPath = s.Path
	m.playingDone = done
	m.setStatus("")
	return m, waitForPlayback(done)
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
		sampleItems[i] = sampleItem{Sample: s, Display: s.Name}
	}

	m.dir = dir
	m.setStatus("")
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

	chrome := headerLines + panelChromeLines*2 + blankLines*blankGaps + padsBoxLines + metadataBoxLines + footerLines
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

	_, halfWidth := m.panelWidths()
	m.folders.SetSize(halfWidth, foldersHeight)
	m.files.SetSize(m.width, filesHeight)
}

// panelWidths returns the content width (excluding border/padding) for a
// full-width panel and for one of the two side-by-side panels in the top
// row (Folders and MIDI telemetry).
func (m Model) panelWidths() (full, half int) {
	full = m.width - 4
	const gap = 1
	half = (m.width-gap)/2 - 4
	if half < 10 {
		half = 10
	}
	return full, half
}

const (
	minTerminalWidth  = 45 // must fit the banner's widest line (41 cols)
	minTerminalHeight = 32
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
	fmt.Fprintln(&b, titleStyle.Render(oscillBanner))
	fmt.Fprintln(&b, pathStyle.Render(rel))
	fmt.Fprintln(&b)

	filesHeader := "Samples"
	if m.searching {
		filesHeader = "Search results"
	}
	if m.focus == focusFolders {
		filesHeader = sectionStyle.Render(filesHeader)
	} else {
		filesHeader = focusedSectionStyle.Render(filesHeader)
	}

	panelWidth, halfWidth := m.panelWidths()

	var foldersBox string
	if m.searching {
		foldersBox = panelStyle(true).Width(halfWidth).
			Render(focusedSectionStyle.Render("Search") + "\n" + m.searchPanelView())
	} else {
		foldersHeader := "Folders"
		if m.focus == focusFolders {
			foldersHeader = focusedSectionStyle.Render(foldersHeader)
		} else {
			foldersHeader = sectionStyle.Render(foldersHeader)
		}
		foldersBox = panelStyle(m.focus == focusFolders).Width(halfWidth).
			Render(foldersHeader + "\n" + m.folders.View())
	}

	midiBox := panelStyle(false).Width(halfWidth).
		Render(sectionStyle.Render("MIDI") + "\n" + m.midiPanelView(m.folders.Height(), halfWidth))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().MarginRight(1).Render(foldersBox), midiBox)

	filesBox := panelStyle(m.focus == focusFiles).Width(panelWidth).
		Render(filesHeader + "\n" + m.files.View())

	padsBox := panelStyle(false).Width(panelWidth).
		Render(sectionStyle.Render("Pads") + "\n" + m.padsView())

	fmt.Fprintln(&b, topRow)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, filesBox)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, padsBox)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, metadataBoxStyle.Width(panelWidth).Render(m.metadataView()))
	fmt.Fprintln(&b)

	help := "↑/↓ move  tab pane  enter open  bkspc up  space play  / search  a assign  m midi  q quit"
	switch {
	case m.status != "":
		fmt.Fprint(&b, statusStyle.Render(m.status))
	case m.playing:
		fmt.Fprint(&b, playingStyle.Render(fmt.Sprintf("▶ Playing: %s   (space to stop)", filepath.Base(m.playingPath))))
	default:
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
	s := item.(sampleItem).Sample

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
