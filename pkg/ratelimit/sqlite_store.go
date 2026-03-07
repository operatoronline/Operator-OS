package ratelimit

import (
	"database/sql"
	"fmt"
	"time"
)

// SQLiteRateLimitStore implements RateLimitStore using SQLite.
type SQLiteRateLimitStore struct {
	db *sql.DB
}

// NewSQLiteRateLimitStore creates a new SQLite-backed rate limit store.
// It creates the rate_limits table if it doesn't exist.
func NewSQLiteRateLimitStore(db *sql.DB) (*SQLiteRateLimitStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db must not be nil")
	}

	if err := initRateLimitSchema(db); err != nil {
		return nil, fmt.Errorf("init rate limit schema: %w", err)
	}

	return &SQLiteRateLimitStore{db: db}, nil
}

func initRateLimitSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS rate_limits (
			user_id    TEXT PRIMARY KEY,
			tier       TEXT NOT NULL DEFAULT 'free',
			tokens     REAL NOT NULL DEFAULT 0,
			last_time  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			day_count  INTEGER NOT NULL DEFAULT 0,
			day_start  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// SaveBucket persists a user's bucket state.
func (s *SQLiteRateLimitStore) SaveBucket(userID string, tier PlanTier, state BucketState) error {
	_, err := s.db.Exec(`
		INSERT INTO rate_limits (user_id, tier, tokens, last_time, day_count, day_start, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			tier = excluded.tier,
			tokens = excluded.tokens,
			last_time = excluded.last_time,
			day_count = excluded.day_count,
			day_start = excluded.day_start,
			updated_at = CURRENT_TIMESTAMP
	`, userID, string(tier), state.Tokens, state.LastTime.UTC().Format(time.RFC3339Nano),
		state.DayCount, state.DayStart.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save rate limit bucket: %w", err)
	}
	return nil
}

// LoadBucket retrieves a user's saved bucket state.
func (s *SQLiteRateLimitStore) LoadBucket(userID string) (PlanTier, BucketState, error) {
	var (
		tierStr   string
		tokens    float64
		lastTime  string
		dayCount  int64
		dayStart  string
	)

	err := s.db.QueryRow(`
		SELECT tier, tokens, last_time, day_count, day_start
		FROM rate_limits WHERE user_id = ?
	`, userID).Scan(&tierStr, &tokens, &lastTime, &dayCount, &dayStart)
	if err == sql.ErrNoRows {
		return "", BucketState{}, ErrNotFound
	}
	if err != nil {
		return "", BucketState{}, fmt.Errorf("load rate limit bucket: %w", err)
	}

	lt, err := time.Parse(time.RFC3339Nano, lastTime)
	if err != nil {
		lt = time.Now()
	}
	ds, err := time.Parse(time.RFC3339Nano, dayStart)
	if err != nil {
		ds = time.Now().Truncate(24 * time.Hour)
	}

	return PlanTier(tierStr), BucketState{
		Tokens:   tokens,
		LastTime: lt,
		DayCount: dayCount,
		DayStart: ds,
	}, nil
}

// DeleteBucket removes a user's saved bucket state.
func (s *SQLiteRateLimitStore) DeleteBucket(userID string) error {
	_, err := s.db.Exec(`DELETE FROM rate_limits WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete rate limit bucket: %w", err)
	}
	return nil
}

// Close is a no-op — the database connection is managed externally.
func (s *SQLiteRateLimitStore) Close() error {
	return nil
}
