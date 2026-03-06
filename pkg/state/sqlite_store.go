package state

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStateStore implements StateStore backed by a SQLite database.
// All writes are immediate (write-through).
type SQLiteStateStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteStateStore opens (or creates) a SQLite database at dbPath and
// initialises the state schema. Uses WAL mode for concurrent read performance.
func NewSQLiteStateStore(dbPath string) (*SQLiteStateStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(4)

	if err := initStateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init state schema: %w", err)
	}

	return &SQLiteStateStore{db: db}, nil
}

func initStateSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS state (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL DEFAULT '',
    updated_at  TEXT NOT NULL DEFAULT (strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now'))
);
`
	_, err := db.Exec(schema)
	return err
}

// Get retrieves the value for the given key. Returns "" if not found.
func (s *SQLiteStateStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value string
	err := s.db.QueryRow(`SELECT value FROM state WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get state %q: %w", key, err)
	}
	return value, nil
}

// Set stores a key-value pair with the current timestamp.
func (s *SQLiteStateStore) Set(key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339Nano)
	_, err := s.db.Exec(
		`INSERT INTO state (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("set state %q: %w", key, err)
	}
	return nil
}

// GetTimestamp returns the last update time for the given key.
// Returns zero time if the key has never been set.
func (s *SQLiteStateStore) GetTimestamp(key string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var updatedStr string
	err := s.db.QueryRow(`SELECT updated_at FROM state WHERE key = ?`, key).Scan(&updatedStr)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get timestamp %q: %w", key, err)
	}

	t, err := time.Parse(time.RFC3339Nano, updatedStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", updatedStr, err)
	}
	return t, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStateStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
