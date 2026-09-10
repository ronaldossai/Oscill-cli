package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func buildTestLibrary(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mustWrite := func(rel string, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	mustWrite("Kicks/kick_808.wav", "not a real wav, just for listing")
	mustWrite("Kicks/kick_deep.wav", "not a real wav, just for listing")
	mustWrite("Snares/snare_tight.wav", "not a real wav, just for listing")
	mustWrite("root_loop.wav", "not a real wav, just for listing")

	return root
}

func sendKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	var msg tea.KeyMsg
	switch key {
	case "up", "down", "left", "right", "enter", "backspace", "tab", "space":
		msg = tea.KeyMsg{Type: keyType(key)}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	return next.(Model)
}

func keyType(key string) tea.KeyType {
	switch key {
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "left":
		return tea.KeyLeft
	case "right":
		return tea.KeyRight
	case "enter":
		return tea.KeyEnter
	case "backspace":
		return tea.KeyBackspace
	case "tab":
		return tea.KeyTab
	case "space":
		return tea.KeySpace
	}
	panic("unknown key " + key)
}

func TestBrowserNavigatesFoldersAndBack(t *testing.T) {
	root := buildTestLibrary(t)

	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	view := m.View()
	if !strings.Contains(view, "Kicks/") || !strings.Contains(view, "Snares/") {
		t.Fatalf("expected top-level folders in view, got:\n%s", view)
	}
	if !strings.Contains(view, "root_loop.wav") {
		t.Fatalf("expected root-level sample in view, got:\n%s", view)
	}

	m = sendKey(t, m, "enter") // descend into the selected folder (Kicks, first alphabetically)
	if filepath.Base(m.dir) != "Kicks" {
		t.Fatalf("dir = %s, want to have descended into Kicks", m.dir)
	}

	view = m.View()
	if !strings.Contains(view, "kick_808.wav") || !strings.Contains(view, "kick_deep.wav") {
		t.Fatalf("expected Kicks samples in view, got:\n%s", view)
	}

	m = sendKey(t, m, "backspace")
	if m.dir != root {
		t.Fatalf("dir = %s, want %s after navigating up", m.dir, root)
	}

	m = sendKey(t, m, "backspace") // already at root, should just set a status
	if m.dir != root {
		t.Fatalf("dir changed above root: %s", m.dir)
	}
	if m.status == "" {
		t.Error("expected a status message when trying to go above the library root")
	}
}

func TestBrowserFocusSwitchAndMetadata(t *testing.T) {
	root := buildTestLibrary(t)

	m, err := NewBrowser(root)
	if err != nil {
		t.Fatalf("NewBrowser: %v", err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 36})
	m = next.(Model)

	if m.focus != focusFolders {
		t.Fatalf("initial focus = %v, want focusFolders", m.focus)
	}

	m = sendKey(t, m, "tab")
	if m.focus != focusFiles {
		t.Fatalf("focus after tab = %v, want focusFiles", m.focus)
	}

	view := m.View()
	if !strings.Contains(view, "root_loop.wav") {
		t.Fatalf("expected selected sample's metadata in view, got:\n%s", view)
	}
}
