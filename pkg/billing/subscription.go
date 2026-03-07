package billing

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SubscriptionStatus represents the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusTrialing  SubscriptionStatus = "trialing"
	SubStatusPastDue   SubscriptionStatus = "past_due"
	SubStatusCanceled  SubscriptionStatus = "canceled"
	SubStatusExpired   SubscriptionStatus = "expired"
	SubStatusPaused    SubscriptionStatus = "paused"
)

// ValidSubscriptionStatus reports whether s is a known status.
func ValidSubscriptionStatus(s SubscriptionStatus) bool {
	switch s {
	case SubStatusActive, SubStatusTrialing, SubStatusPastDue,
		SubStatusCanceled, SubStatusExpired, SubStatusPaused:
		return true
	}
	return false
}

// Subscription tracks a user's plan assignment and billing state.
type Subscription struct {
	ID                 string             `json:"id"`
	UserID             string             `json:"user_id"`
	PlanID             PlanID             `json:"plan_id"`
	Status             SubscriptionStatus `json:"status"`
	BillingInterval    BillingInterval    `json:"billing_interval"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
	StripeCustomerID   string             `json:"stripe_customer_id,omitempty"`
	StripeSubID        string             `json:"stripe_subscription_id,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// IsActive reports whether the subscription currently grants access.
func (s *Subscription) IsActive() bool {
	return s.Status == SubStatusActive || s.Status == SubStatusTrialing || s.Status == SubStatusPastDue
}

// SubscriptionStore abstracts subscription persistence.
type SubscriptionStore interface {
	// Create inserts a new subscription. Returns an error if a duplicate ID exists.
	Create(sub *Subscription) error
	// GetByID returns the subscription with the given ID, or ErrNotFound.
	GetByID(id string) (*Subscription, error)
	// GetByUserID returns the most recent subscription for a user, or ErrNotFound.
	GetByUserID(userID string) (*Subscription, error)
	// Update saves changes to an existing subscription.
	Update(sub *Subscription) error
	// ListByStatus returns all subscriptions with the given status.
	ListByStatus(status SubscriptionStatus) ([]*Subscription, error)
	// Close releases resources.
	Close() error
}

// Common errors.
var (
	ErrNotFound      = fmt.Errorf("billing: not found")
	ErrDuplicateID   = fmt.Errorf("billing: duplicate id")
	ErrInvalidStatus = fmt.Errorf("billing: invalid status")
	ErrInvalidPlan   = fmt.Errorf("billing: invalid plan")
)

// ---------- SQLite implementation ----------

const createSubscriptionsSQL = `
CREATE TABLE IF NOT EXISTS subscriptions (
	id                    TEXT PRIMARY KEY,
	user_id               TEXT NOT NULL,
	plan_id               TEXT NOT NULL DEFAULT 'free',
	status                TEXT NOT NULL DEFAULT 'active',
	billing_interval      TEXT NOT NULL DEFAULT 'none',
	current_period_start  DATETIME NOT NULL,
	current_period_end    DATETIME NOT NULL,
	cancel_at_period_end  INTEGER NOT NULL DEFAULT 0,
	stripe_customer_id    TEXT DEFAULT '',
	stripe_subscription_id TEXT DEFAULT '',
	created_at            DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at            DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status  ON subscriptions(status);
`

// SQLiteSubscriptionStore implements SubscriptionStore backed by SQLite.
type SQLiteSubscriptionStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// NewSQLiteSubscriptionStore creates a SQLiteSubscriptionStore and ensures the table exists.
func NewSQLiteSubscriptionStore(db *sql.DB) (*SQLiteSubscriptionStore, error) {
	if db == nil {
		return nil, fmt.Errorf("billing: db is nil")
	}
	if _, err := db.Exec(createSubscriptionsSQL); err != nil {
		return nil, fmt.Errorf("billing: create subscriptions table: %w", err)
	}
	return &SQLiteSubscriptionStore{db: db}, nil
}

func (s *SQLiteSubscriptionStore) Create(sub *Subscription) error {
	if sub == nil {
		return fmt.Errorf("billing: subscription is nil")
	}
	if sub.ID == "" {
		return fmt.Errorf("billing: subscription ID is empty")
	}
	if !ValidPlanID(sub.PlanID) {
		return ErrInvalidPlan
	}
	if sub.Status == "" {
		sub.Status = SubStatusActive
	}
	if !ValidSubscriptionStatus(sub.Status) {
		return ErrInvalidStatus
	}

	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (
			id, user_id, plan_id, status, billing_interval,
			current_period_start, current_period_end, cancel_at_period_end,
			stripe_customer_id, stripe_subscription_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID, sub.UserID, string(sub.PlanID), string(sub.Status), string(sub.BillingInterval),
		sub.CurrentPeriodStart.UTC(), sub.CurrentPeriodEnd.UTC(), boolToInt(sub.CancelAtPeriodEnd),
		sub.StripeCustomerID, sub.StripeSubID, sub.CreatedAt.UTC(), sub.UpdatedAt.UTC(),
	)
	if err != nil {
		if isDuplicateErr(err) {
			return ErrDuplicateID
		}
		return fmt.Errorf("billing: insert subscription: %w", err)
	}
	return nil
}

func (s *SQLiteSubscriptionStore) GetByID(id string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanOne(`SELECT * FROM subscriptions WHERE id = ?`, id)
}

func (s *SQLiteSubscriptionStore) GetByUserID(userID string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanOne(`SELECT * FROM subscriptions WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`, userID)
}

func (s *SQLiteSubscriptionStore) Update(sub *Subscription) error {
	if sub == nil {
		return fmt.Errorf("billing: subscription is nil")
	}
	if !ValidSubscriptionStatus(sub.Status) {
		return ErrInvalidStatus
	}
	if !ValidPlanID(sub.PlanID) {
		return ErrInvalidPlan
	}

	sub.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
		UPDATE subscriptions SET
			plan_id = ?, status = ?, billing_interval = ?,
			current_period_start = ?, current_period_end = ?,
			cancel_at_period_end = ?,
			stripe_customer_id = ?, stripe_subscription_id = ?,
			updated_at = ?
		WHERE id = ?`,
		string(sub.PlanID), string(sub.Status), string(sub.BillingInterval),
		sub.CurrentPeriodStart.UTC(), sub.CurrentPeriodEnd.UTC(),
		boolToInt(sub.CancelAtPeriodEnd),
		sub.StripeCustomerID, sub.StripeSubID,
		sub.UpdatedAt.UTC(), sub.ID,
	)
	if err != nil {
		return fmt.Errorf("billing: update subscription: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteSubscriptionStore) ListByStatus(status SubscriptionStatus) ([]*Subscription, error) {
	if !ValidSubscriptionStatus(status) {
		return nil, ErrInvalidStatus
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT * FROM subscriptions WHERE status = ? ORDER BY created_at DESC`, string(status))
	if err != nil {
		return nil, fmt.Errorf("billing: list subscriptions: %w", err)
	}
	defer rows.Close()

	var out []*Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *SQLiteSubscriptionStore) Close() error { return nil }

// scanOne runs a query expecting exactly one row.
func (s *SQLiteSubscriptionStore) scanOne(query string, args ...any) (*Subscription, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("billing: query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNotFound
	}
	sub, err := scanSubscription(rows)
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		// should not happen but consume
	}
	return sub, rows.Err()
}

// scanSubscription reads a Subscription from the current row.
func scanSubscription(rows *sql.Rows) (*Subscription, error) {
	var sub Subscription
	var planID, status, interval string
	var cancelInt int

	err := rows.Scan(
		&sub.ID, &sub.UserID, &planID, &status, &interval,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &cancelInt,
		&sub.StripeCustomerID, &sub.StripeSubID,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("billing: scan subscription: %w", err)
	}

	sub.PlanID = PlanID(planID)
	sub.Status = SubscriptionStatus(status)
	sub.BillingInterval = BillingInterval(interval)
	sub.CancelAtPeriodEnd = cancelInt != 0
	return &sub, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isDuplicateErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
