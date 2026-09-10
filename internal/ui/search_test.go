package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	return m
}

func TestSearchFiltersAcrossFolders(t *testing.T) {
	root := buildTestLibrary(t)
	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	if !m.searching || !m.searchTyping {
		t.Fatal("expected searching = true, searchTyping = true after pressing /")
	}
	if cmd == nil {
		t.Fatal("expected a command (cursor blink) after starting search")
	}

	// Before typing anything, every sample in the library should be listed.
	if got := len(m.files.Items()); got != 4 {
		t.Fatalf("initial search results = %d, want 4 (all samples)", got)
	}

	m = typeText(t, m, "kick")

	items := m.files.Items()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 matches for \"kick\"", len(items))
	}
	for _, it := range items {
		si := it.(sampleItem)
		if !strings.Contains(strings.ToLower(si.Sample.Name), "kick") {
			t.Errorf("unexpected match %s for query \"kick\"", si.Sample.Name)
		}
		if si.Display == si.Sample.Name {
			t.Errorf("expected a folder-qualified Display for a search result, got bare name %q", si.Display)
		}
	}

	view := m.View()
	if !strings.Contains(view, "kick") {
		t.Errorf("expected query text in the view, got:\n%s", view)
	}
}

func TestSearchConfirmThenNavigate(t *testing.T) {
	root := buildTestLibrary(t)
	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	m = typeText(t, m, "kick")

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !m.searching || m.searchTyping {
		t.Fatal("expected searching = true, searchTyping = false after confirming with enter")
	}

	// Now that we're out of typing mode, arrow keys should move the list
	// selection rather than being typed into the query.
	m = sendKey(t, m, "down")
	if m.searchInput.Value() != "kick" {
		t.Errorf("query changed after confirming: %q", m.searchInput.Value())
	}

	selected := m.files.SelectedItem().(sampleItem)

	// Jump to the selected result: this should load its containing folder
	// and exit search mode entirely.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)

	if m.searching {
		t.Fatal("expected searching = false after jumping to a result")
	}
	if m.dir != filepath.Dir(selected.Sample.Path) {
		t.Fatalf("dir = %s, want %s", m.dir, filepath.Dir(selected.Sample.Path))
	}

	jumped, ok := m.files.SelectedItem().(sampleItem)
	if !ok || jumped.Sample.Path != selected.Sample.Path {
		t.Errorf("expected the jumped-to folder to have the same sample selected")
	}
}

func TestSearchEscClearsAndRestoresListing(t *testing.T) {
	root := buildTestLibrary(t)
	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	originalItems := len(m.files.Items())

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	m = typeText(t, m, "kick")

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)

	if m.searching || m.searchTyping {
		t.Fatal("expected search state fully cleared after esc")
	}
	if got := len(m.files.Items()); got != originalItems {
		t.Fatalf("Samples pane has %d items after clearing search, want %d (back to normal dir listing)", got, originalItems)
	}
}

func TestTabDisabledWhileSearching(t *testing.T) {
	root := buildTestLibrary(t)
	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm, exit typing
	m = next.(Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if m.focus != focusFiles {
		t.Errorf("focus = %v, want focusFiles (tab should be a no-op while searching)", m.focus)
	}
}
