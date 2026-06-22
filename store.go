package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type trackedSession struct {
	ProjectName string
	StartedAt   time.Time
	EndedAt     time.Time
	Duration    time.Duration
}

type sessionStore interface {
	RecordSession(context.Context, trackedSession) error
	ExportCSV(context.Context, string) (int, error)
	Close() error
}

type sqliteSessionStore struct {
	db *sql.DB
}

func openSQLiteSessionStore(path string) (*sqliteSessionStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	store := &sqliteSessionStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *sqliteSessionStore) migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_name TEXT NOT NULL,
	started_at TEXT NOT NULL,
	ended_at TEXT NOT NULL,
	duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);
`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}
	return nil
}

func (s *sqliteSessionStore) RecordSession(ctx context.Context, session trackedSession) error {
	if s == nil || s.db == nil {
		return errors.New("session store is not open")
	}
	if session.ProjectName == "" {
		return errors.New("project name is required")
	}
	if session.Duration <= 0 {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (project_name, started_at, ended_at, duration_ms, created_at)
VALUES (?, ?, ?, ?, ?)
`, session.ProjectName,
		session.StartedAt.UTC().Format(time.RFC3339Nano),
		session.EndedAt.UTC().Format(time.RFC3339Nano),
		session.Duration.Milliseconds(),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record session: %w", err)
	}
	return nil
}

func (s *sqliteSessionStore) ExportCSV(ctx context.Context, path string) (int, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("session store is not open")
	}
	if path == "" {
		return 0, errors.New("csv path is required")
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_name, started_at, ended_at, duration_ms, created_at
FROM sessions
ORDER BY started_at, id
`)
	if err != nil {
		return 0, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	file, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create csv file: %w", err)
	}

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{
		"id",
		"project_name",
		"started_at",
		"ended_at",
		"duration_ms",
		"duration_seconds",
		"duration_hhmmss",
		"created_at",
	}); err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("write csv header: %w", err)
	}

	count := 0
	for rows.Next() {
		var (
			id          int64
			projectName string
			startedAt   string
			endedAt     string
			durationMS  int64
			createdAt   string
		)
		if err := rows.Scan(&id, &projectName, &startedAt, &endedAt, &durationMS, &createdAt); err != nil {
			_ = file.Close()
			return count, fmt.Errorf("scan session: %w", err)
		}

		duration := time.Duration(durationMS) * time.Millisecond
		record := []string{
			strconv.FormatInt(id, 10),
			projectName,
			startedAt,
			endedAt,
			strconv.FormatInt(durationMS, 10),
			strconv.FormatFloat(float64(durationMS)/1000, 'f', 3, 64),
			formatDuration(duration),
			createdAt,
		}
		if err := writer.Write(record); err != nil {
			_ = file.Close()
			return count, fmt.Errorf("write csv record: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		_ = file.Close()
		return count, fmt.Errorf("read sessions: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return count, fmt.Errorf("flush csv file: %w", err)
	}
	if err := file.Close(); err != nil {
		return count, fmt.Errorf("close csv file: %w", err)
	}

	return count, nil
}

func (s *sqliteSessionStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
