package ui

import (
	"testing"
	"time"
)

func TestStatusAutoClearsAfterTimeout(t *testing.T) {
	m, _ := newTestBrowser(t)
	m.setStatus("already at the library root")

	// Not aged past the timeout yet: a tick should leave it alone.
	next, cmd := m.Update(statusTickMsg{})
	m = next.(Model)
	if m.status == "" {
		t.Fatal("status cleared before statusTimeout elapsed")
	}
	if cmd == nil {
		t.Fatal("expected statusTickMsg to re-arm the check loop")
	}

	// Simulate the timeout having elapsed.
	m.statusSetAt = time.Now().Add(-statusTimeout - time.Second)
	next, _ = m.Update(statusTickMsg{})
	m = next.(Model)
	if m.status != "" {
		t.Fatalf("status = %q, want cleared after statusTimeout elapsed", m.status)
	}
}

func TestStatusDoesNotAutoClearDuringActiveMode(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Model)
	}{
		{"assigning", func(m *Model) { m.assigning = true }},
		{"learning", func(m *Model) { m.learning = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := newTestBrowser(t)
			m.setStatus("mid-mode reminder")
			tt.setup(&m)
			m.statusSetAt = time.Now().Add(-statusTimeout - time.Second)

			next, _ := m.Update(statusTickMsg{})
			m = next.(Model)
			if m.status == "" {
				t.Error("status was cleared while an active mode was in progress")
			}
		})
	}
}

func TestSetStatusClearsImmediately(t *testing.T) {
	var m Model
	m.setStatus("something")
	if m.status == "" {
		t.Fatal("setStatus did not set the message")
	}
	m.setStatus("")
	if m.status != "" {
		t.Fatalf("status = %q, want empty after setStatus(\"\")", m.status)
	}
}
