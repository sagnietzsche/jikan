package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestToggleSelectedAccumulatesElapsed(t *testing.T) {
	start := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	m := newModelAt(start)

	m.toggleSelected()
	m.now = start.Add(90 * time.Second)
	m.toggleSelected()

	if got := m.elapsedFor(0); got != 90*time.Second {
		t.Fatalf("elapsedFor(0) = %s, want 1m30s", got)
	}
	if m.active != -1 {
		t.Fatalf("active = %d, want -1 after stopping", m.active)
	}
}

func TestSwitchingProjectsStopsPreviousTimer(t *testing.T) {
	start := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	m := newModelAt(start)

	m.toggleSelected()
	m.now = start.Add(45 * time.Second)
	m.moveCursor(1)
	m.toggleSelected()
	m.now = start.Add(75 * time.Second)

	if got := m.elapsedFor(0); got != 45*time.Second {
		t.Fatalf("elapsedFor(0) = %s, want 45s", got)
	}
	if got := m.elapsedFor(1); got != 30*time.Second {
		t.Fatalf("elapsedFor(1) = %s, want 30s", got)
	}
	if m.active != 1 {
		t.Fatalf("active = %d, want 1", m.active)
	}
}

func TestAddProjectThroughInput(t *testing.T) {
	m := newModelAt(time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC))
	m.creating = true
	m.input.SetValue("  Planning Notes  ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if got := m.projects[len(m.projects)-1].Name; got != "Planning Notes" {
		t.Fatalf("new project name = %q, want %q", got, "Planning Notes")
	}
	if m.creating {
		t.Fatal("creating = true, want false after saving")
	}
	if m.cursor != len(m.projects)-1 {
		t.Fatalf("cursor = %d, want last project", m.cursor)
	}
}

func TestFormatDuration(t *testing.T) {
	got := formatDuration(25*time.Hour + 3*time.Minute + 7*time.Second + 800*time.Millisecond)
	if got != "25:03:07" {
		t.Fatalf("formatDuration() = %q, want %q", got, "25:03:07")
	}
}

func TestViewIncludesEnglishBranding(t *testing.T) {
	m := newModelAt(time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC))
	view := m.View()

	for _, want := range []string{"time", "jikan", "Morning Planning"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q\n%s", want, view)
		}
	}
}
