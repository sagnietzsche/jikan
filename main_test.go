package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeSessionStore struct {
	sessions []trackedSession
}

func (s *fakeSessionStore) RecordSession(_ context.Context, session trackedSession) error {
	s.sessions = append(s.sessions, session)
	return nil
}

func (s *fakeSessionStore) ExportCSV(_ context.Context, _ string) (int, error) {
	return len(s.sessions), nil
}

func (s *fakeSessionStore) Close() error {
	return nil
}

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

func TestStopActiveRecordsSession(t *testing.T) {
	start := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	store := &fakeSessionStore{}
	m := newModelWithStore(start, store)

	m.toggleSelected()
	m.now = start.Add(90 * time.Second)
	m.toggleSelected()

	if len(store.sessions) != 1 {
		t.Fatalf("recorded sessions = %d, want 1", len(store.sessions))
	}
	got := store.sessions[0]
	if got.ProjectName != "Morning Planning" {
		t.Fatalf("project name = %q, want Morning Planning", got.ProjectName)
	}
	if !got.StartedAt.Equal(start) {
		t.Fatalf("started at = %s, want %s", got.StartedAt, start)
	}
	if !got.EndedAt.Equal(start.Add(90 * time.Second)) {
		t.Fatalf("ended at = %s, want %s", got.EndedAt, start.Add(90*time.Second))
	}
	if got.Duration != 90*time.Second {
		t.Fatalf("duration = %s, want 1m30s", got.Duration)
	}
	if !strings.Contains(m.status, "saved") {
		t.Fatalf("status = %q, want saved message", m.status)
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

func TestSQLiteStoreRecordsAndExportsCSV(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "jikan.db")
	csvPath := filepath.Join(dir, "sessions.csv")
	start := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)

	store, err := openSQLiteSessionStore(dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	err = store.RecordSession(context.Background(), trackedSession{
		ProjectName: "Deep Work",
		StartedAt:   start,
		EndedAt:     start.Add(90 * time.Second),
		Duration:    90 * time.Second,
	})
	if err != nil {
		t.Fatalf("record session: %v", err)
	}

	count, err := store.ExportCSV(context.Background(), csvPath)
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if count != 1 {
		t.Fatalf("exported rows = %d, want 1", count)
	}

	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	csv := string(data)
	for _, want := range []string{
		"id,project_name,started_at,ended_at,duration_ms,duration_seconds,duration_hhmmss,created_at",
		"Deep Work",
		"90000",
		"90.000",
		"00:01:30",
	} {
		if !strings.Contains(csv, want) {
			t.Fatalf("csv missing %q\n%s", want, csv)
		}
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

	for _, want := range []string{"time", "jikan", "Morning Planning", "e export"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q\n%s", want, view)
		}
	}
}
