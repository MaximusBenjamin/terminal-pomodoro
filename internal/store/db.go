package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home dir: %w", err)
	}

	dir := filepath.Join(home, ".pomo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dir, "pomo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS habits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#7aa2f7',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		archived INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		habit_id INTEGER NOT NULL REFERENCES habits(id),
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		planned_minutes INTEGER NOT NULL DEFAULT 25,
		actual_seconds INTEGER DEFAULT 0,
		overtime_seconds INTEGER DEFAULT 0,
		completed INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_start ON sessions(start_time);
	CREATE INDEX IF NOT EXISTS idx_sessions_habit ON sessions(habit_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("creating tables: %w", err)
	}

	// Seed default habits if none exist
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM habits").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		defaults := []struct {
			name  string
			color string
		}{
			{"programming", "#7aa2f7"},
			{"mathematics", "#bb9af7"},
			{"finance", "#9ece6a"},
			{"reading", "#e0af68"},
			{"writing", "#7dcfff"},
		}
		for _, h := range defaults {
			if _, err := s.db.Exec("INSERT INTO habits (name, color) VALUES (?, ?)", h.name, h.color); err != nil {
				return fmt.Errorf("seeding habit %s: %w", h.name, err)
			}
		}
	}

	return nil
}
