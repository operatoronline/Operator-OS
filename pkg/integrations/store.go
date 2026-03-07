package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/standardws/operator/pkg/logger"
)

// UserIntegration tracks the status of an integration for a specific user.
type UserIntegration struct {
	// ID is the unique record identifier.
	ID string `json:"id"`
	// UserID is the owning user.
	UserID string `json:"user_id"`
	// IntegrationID references the Integration manifest ID (e.g. "google", "shopify").
	IntegrationID string `json:"integration_id"`
	// Status is the lifecycle state.
	Status string `json:"status"`
	// Config holds integration-specific configuration (e.g. shop URL for Shopify).
	Config map[string]string `json:"config,omitempty"`
	// Scopes are the granted OAuth scopes.
	Scopes []string `json:"scopes,omitempty"`
	// ErrorMessage stores the last error if status is "failed".
	ErrorMessage string `json:"error_message,omitempty"`
	// LastUsedAt is the last time this integration was used.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// CreatedAt is when this integration was connected.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when this record was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// User integration statuses.
const (
	UserIntegrationPending  = "pending"
	UserIntegrationActive   = "active"
	UserIntegrationFailed   = "failed"
	UserIntegrationRevoked  = "revoked"
	UserIntegrationDisabled = "disabled"
)

// ValidUserIntegrationStatus checks if the status is recognized.
func ValidUserIntegrationStatus(s string) bool {
	switch s {
	case UserIntegrationPending, UserIntegrationActive, UserIntegrationFailed,
		UserIntegrationRevoked, UserIntegrationDisabled:
		return true
	}
	return false
}

// UserIntegrationStore persists per-user integration state.
type UserIntegrationStore interface {
	// Create creates a new user integration record.
	Create(ui *UserIntegration) error
	// Get retrieves a user's integration by user ID and integration ID.
	Get(userID, integrationID string) (*UserIntegration, error)
	// GetByID retrieves a user integration by its unique ID.
	GetByID(id string) (*UserIntegration, error)
	// ListByUser returns all integrations for a user, optionally filtered by status.
	ListByUser(userID string, status string) ([]*UserIntegration, error)
	// Update updates an existing user integration record.
	Update(ui *UserIntegration) error
	// UpdateStatus sets the status (and optional error message) for a user integration.
	UpdateStatus(userID, integrationID, status, errorMsg string) error
	// Delete removes a user integration record.
	Delete(userID, integrationID string) error
	// RecordUsage updates the last_used_at timestamp.
	RecordUsage(userID, integrationID string) error
	// CountByUser returns the number of integrations for a user.
	CountByUser(userID string) (int, error)
	// Close closes the store.
	Close() error
}

// SQLiteUserIntegrationStore implements UserIntegrationStore backed by SQLite.
type SQLiteUserIntegrationStore struct {
	db *sql.DB
}

// NewSQLiteUserIntegrationStore creates a new SQLite-backed user integration store.
func NewSQLiteUserIntegrationStore(db *sql.DB) (*SQLiteUserIntegrationStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &SQLiteUserIntegrationStore{db: db}, nil
}

func (s *SQLiteUserIntegrationStore) Create(ui *UserIntegration) error {
	if ui == nil {
		return fmt.Errorf("user integration is nil")
	}
	if ui.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if ui.IntegrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	if ui.Status == "" {
		ui.Status = UserIntegrationPending
	}
	if !ValidUserIntegrationStatus(ui.Status) {
		return fmt.Errorf("invalid status %q", ui.Status)
	}
	if ui.ID == "" {
		ui.ID = generateID()
	}
	now := time.Now().UTC()
	ui.CreatedAt = now
	ui.UpdatedAt = now

	configJSON, err := json.Marshal(ui.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	scopesJSON, err := json.Marshal(ui.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO user_integrations (id, user_id, integration_id, status, config, scopes, error_message, last_used_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ui.ID, ui.UserID, ui.IntegrationID, ui.Status,
		string(configJSON), string(scopesJSON), ui.ErrorMessage,
		nilTime(ui.LastUsedAt), ui.CreatedAt.Format(time.RFC3339Nano), ui.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("integration %q already connected for user", ui.IntegrationID)
		}
		return fmt.Errorf("failed to create user integration: %w", err)
	}
	return nil
}

func (s *SQLiteUserIntegrationStore) Get(userID, integrationID string) (*UserIntegration, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if integrationID == "" {
		return nil, fmt.Errorf("integration_id is required")
	}
	row := s.db.QueryRow(`
		SELECT id, user_id, integration_id, status, config, scopes, error_message, last_used_at, created_at, updated_at
		FROM user_integrations WHERE user_id = ? AND integration_id = ?`, userID, integrationID)
	return scanUserIntegration(row)
}

func (s *SQLiteUserIntegrationStore) GetByID(id string) (*UserIntegration, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	row := s.db.QueryRow(`
		SELECT id, user_id, integration_id, status, config, scopes, error_message, last_used_at, created_at, updated_at
		FROM user_integrations WHERE id = ?`, id)
	return scanUserIntegration(row)
}

func (s *SQLiteUserIntegrationStore) ListByUser(userID string, status string) ([]*UserIntegration, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	var rows *sql.Rows
	var err error
	if status != "" {
		if !ValidUserIntegrationStatus(status) {
			return nil, fmt.Errorf("invalid status filter %q", status)
		}
		rows, err = s.db.Query(`
			SELECT id, user_id, integration_id, status, config, scopes, error_message, last_used_at, created_at, updated_at
			FROM user_integrations WHERE user_id = ? AND status = ? ORDER BY created_at ASC`, userID, status)
	} else {
		rows, err = s.db.Query(`
			SELECT id, user_id, integration_id, status, config, scopes, error_message, last_used_at, created_at, updated_at
			FROM user_integrations WHERE user_id = ? ORDER BY created_at ASC`, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list user integrations: %w", err)
	}
	defer rows.Close()

	var result []*UserIntegration
	for rows.Next() {
		ui, err := scanUserIntegrationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, ui)
	}
	return result, rows.Err()
}

func (s *SQLiteUserIntegrationStore) Update(ui *UserIntegration) error {
	if ui == nil {
		return fmt.Errorf("user integration is nil")
	}
	if ui.ID == "" {
		return fmt.Errorf("id is required")
	}
	if ui.Status != "" && !ValidUserIntegrationStatus(ui.Status) {
		return fmt.Errorf("invalid status %q", ui.Status)
	}
	ui.UpdatedAt = time.Now().UTC()

	configJSON, err := json.Marshal(ui.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	scopesJSON, err := json.Marshal(ui.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE user_integrations SET status = ?, config = ?, scopes = ?, error_message = ?, last_used_at = ?, updated_at = ?
		WHERE id = ?`,
		ui.Status, string(configJSON), string(scopesJSON), ui.ErrorMessage,
		nilTime(ui.LastUsedAt), ui.UpdatedAt.Format(time.RFC3339Nano), ui.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user integration: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user integration not found")
	}
	return nil
}

func (s *SQLiteUserIntegrationStore) UpdateStatus(userID, integrationID, status, errorMsg string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if integrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	if !ValidUserIntegrationStatus(status) {
		return fmt.Errorf("invalid status %q", status)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.Exec(`
		UPDATE user_integrations SET status = ?, error_message = ?, updated_at = ?
		WHERE user_id = ? AND integration_id = ?`,
		status, errorMsg, now, userID, integrationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user integration not found")
	}
	return nil
}

func (s *SQLiteUserIntegrationStore) Delete(userID, integrationID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if integrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	result, err := s.db.Exec(`DELETE FROM user_integrations WHERE user_id = ? AND integration_id = ?`, userID, integrationID)
	if err != nil {
		return fmt.Errorf("failed to delete user integration: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user integration not found")
	}
	return nil
}

func (s *SQLiteUserIntegrationStore) RecordUsage(userID, integrationID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if integrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.Exec(`
		UPDATE user_integrations SET last_used_at = ?, updated_at = ?
		WHERE user_id = ? AND integration_id = ?`, now, now, userID, integrationID)
	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user integration not found")
	}
	return nil
}

func (s *SQLiteUserIntegrationStore) CountByUser(userID string) (int, error) {
	if userID == "" {
		return 0, fmt.Errorf("user_id is required")
	}
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM user_integrations WHERE user_id = ?`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user integrations: %w", err)
	}
	return count, nil
}

func (s *SQLiteUserIntegrationStore) Close() error {
	return nil // DB lifecycle managed externally
}

// --- helpers ---

func generateID() string {
	// Use crypto/rand for UUID-like IDs
	b := make([]byte, 16)
	_, err := randRead(b)
	if err != nil {
		logger.WarnCF("integrations", "crypto/rand failed, using timestamp fallback", nil)
		return fmt.Sprintf("ui_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// randRead is a package-var for testability.
var randRead = cryptoRandRead

func cryptoRandRead(b []byte) (int, error) {
	// Import at function level to avoid init-time issues.
	return _cryptoRand(b)
}

func nilTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUserIntegration(row scannable) (*UserIntegration, error) {
	var ui UserIntegration
	var configStr, scopesStr string
	var lastUsedAt sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&ui.ID, &ui.UserID, &ui.IntegrationID, &ui.Status,
		&configStr, &scopesStr, &ui.ErrorMessage, &lastUsedAt,
		&createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user integration not found")
		}
		return nil, fmt.Errorf("failed to scan user integration: %w", err)
	}

	if configStr != "" && configStr != "null" {
		if err := json.Unmarshal([]byte(configStr), &ui.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}
	if scopesStr != "" && scopesStr != "null" {
		if err := json.Unmarshal([]byte(scopesStr), &ui.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}
	}
	if lastUsedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, lastUsedAt.String)
		if err == nil {
			ui.LastUsedAt = &t
		}
	}
	ui.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	ui.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &ui, nil
}

func scanUserIntegrationRow(rows *sql.Rows) (*UserIntegration, error) {
	return scanUserIntegration(rows)
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") ||
		contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
