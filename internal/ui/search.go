package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ronaldossai/Oscill-cli/internal/library"
)

func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "search the whole library by name or folder…"
	ti.CharLimit = 200
	return ti
}

// startSearch enters search mode: it takes a fresh recursive scan of the
// whole library (cached for the rest of the search session, so typing
// filters in memory instead of re-walking the filesystem per keystroke) and
// starts capturing keystrokes into the query.
func (m Model) startSearch() (tea.Model, tea.Cmd) {
	all, err := library.Scan(m.root)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}

	m.searchAll = all
	m.searching = true
	m.searchTyping = true
	m.searchInput = newSearchInput()
	m.focus = focusFiles
	m.status = ""
	m.applySearchResults("")

	return m, m.searchInput.Focus()
}

// clearSearch exits search mode entirely, restoring the current directory's
// normal listing in the Samples pane and the Folders panel.
func (m *Model) clearSearch() {
	m.searching = false
	m.searchTyping = false
	m.searchAll = nil
	m.searchInput = textinput.Model{}
	m.load(m.dir)
}

// applySearchResults filters the cached library scan by query and puts the
// results in the Samples pane, each labeled with its path relative to the
// library root since matches can span multiple folders.
func (m *Model) applySearchResults(query string) {
	matches := library.Filter(m.searchAll, library.Query{Name: query})

	items := make([]list.Item, len(matches))
	for i, s := range matches {
		rel, err := filepath.Rel(m.root, s.Path)
		if err != nil {
			rel = s.Path
		}
		items[i] = sampleItem{Sample: s, Display: rel}
	}
	m.files.SetItems(items)
	m.files.Select(0)
}

// handleSearchTypingKey processes a key while actively typing a search
// query. Enter confirms it (the filtered results stay, but arrow keys etc.
// go back to normal list navigation instead of the text box); esc clears
// the search entirely; everything else is forwarded to the text input.
func (m Model) handleSearchTypingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchTyping = false
		m.searchInput.Blur()
		return m, nil
	case "esc":
		m.clearSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.applySearchResults(m.searchInput.Value())
	return m, cmd
}

// jumpToSearchResult navigates to and highlights the selected search
// result's containing folder, then exits search mode — a way to actually
// use a search hit rather than just look at it.
func (m Model) jumpToSearchResult() (tea.Model, tea.Cmd) {
	item, ok := m.files.SelectedItem().(sampleItem)
	if !ok {
		return m, nil
	}
	s := item.Sample
	dir := filepath.Dir(s.Path)

	m.searching = false
	m.searchTyping = false
	m.searchAll = nil

	if err := m.load(dir); err != nil {
		m.status = err.Error()
		return m, nil
	}
	for i, it := range m.files.Items() {
		if si, ok := it.(sampleItem); ok && si.Sample.Path == s.Path {
			m.files.Select(i)
			break
		}
	}
	return m, nil
}

// searchStatusLine summarizes the current result count.
func (m Model) searchStatusLine() string {
	n := len(m.files.Items())
	plural := "s"
	if n == 1 {
		plural = ""
	}
	return fmt.Sprintf("%d result%s — press / to edit, esc to clear", n, plural)
}

// searchPanelView renders the content that replaces the Folders panel while
// searching, padded to the same height so the layout doesn't shift.
func (m Model) searchPanelView() string {
	lines := []string{m.searchInput.View()}
	if !m.searchTyping {
		lines = append(lines, m.searchStatusLine())
	}
	height := m.folders.Height()
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
