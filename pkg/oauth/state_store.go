package oauth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// DefaultStateTTL is the default lifetime for OAuth state tokens.
const DefaultStateTTL = 10 * time.Minute

// OAuthState represents an in-flight OAuth authorization flow.
type OAuthState struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProviderID   string    `json:"provider_id"`
	State        string    `json:"state"`         // CSRF token
	CodeVerifier string    `json:"code_verifier"` // PKCE code verifier (empty if PKCE not used)
	RedirectURI  string    `json:"redirect_uri"`  // Where to send user after completion
	Scopes       string    `json:"scopes"`        // Requested scopes (space-separated)
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Used         bool      `json:"used"`
}

// TokenResponse is the parsed token response from an OAuth provider.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ProviderID   string    `json:"provider_id"`
	UserID       string    `json:"user_id"`
}

// StateStore persists OAuth authorization state for in-flight flows.
type StateStore interface {
	// Create persists a new OAuth state.
	Create(state *OAuthState) error
	// GetByState retrieves state by the CSRF state token.
	GetByState(stateToken string) (*OAuthState, error)
	// MarkUsed marks a state token as consumed.
	MarkUsed(id string) error
	// DeleteExpired removes expired state entries. Returns count deleted.
	DeleteExpired() (int64, error)
	// Close releases resources.
	Close() error
}

// SQLiteStateStore implements StateStore using SQLite.
type SQLiteStateStore struct {
	db *sql.DB
}

// NewSQLiteStateStore creates a new SQLite-backed state store.
// The caller must ensure the oauth_states table exists (via migration).
func NewSQLiteStateStore(db *sql.DB) (*SQLiteStateStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	return &SQLiteStateStore{db: db}, nil
}

// Create inserts a new OAuth state record.
func (s *SQLiteStateStore) Create(state *OAuthState) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}
	if state.ID == "" {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generating ID: %w", err)
		}
		state.ID = id
	}
	if state.State == "" {
		return fmt.Errorf("state token is required")
	}
	if state.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if state.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now().UTC()
	}
	if state.ExpiresAt.IsZero() {
		state.ExpiresAt = state.CreatedAt.Add(DefaultStateTTL)
	}

	_, err := s.db.Exec(`
		INSERT INTO oauth_states (id, user_id, provider_id, state, code_verifier, redirect_uri, scopes, created_at, expires_at, used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.ID, state.UserID, state.ProviderID, state.State,
		state.CodeVerifier, state.RedirectURI, state.Scopes,
		state.CreatedAt.Format(time.RFC3339Nano),
		state.ExpiresAt.Format(time.RFC3339Nano),
		state.Used,
	)
	if err != nil {
		return fmt.Errorf("inserting oauth state: %w", err)
	}
	return nil
}

// GetByState retrieves an OAuth state by the CSRF state token.
func (s *SQLiteStateStore) GetByState(stateToken string) (*OAuthState, error) {
	if stateToken == "" {
		return nil, fmt.Errorf("state token is required")
	}

	var state OAuthState
	var createdAt, expiresAt string
	err := s.db.QueryRow(`
		SELECT id, user_id, provider_id, state, code_verifier, redirect_uri, scopes, created_at, expires_at, used
		FROM oauth_states WHERE state = ?`, stateToken,
	).Scan(
		&state.ID, &state.UserID, &state.ProviderID, &state.State,
		&state.CodeVerifier, &state.RedirectURI, &state.Scopes,
		&createdAt, &expiresAt, &state.Used,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying oauth state: %w", err)
	}

	state.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	state.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	return &state, nil
}

// MarkUsed marks a state record as consumed.
func (s *SQLiteStateStore) MarkUsed(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	result, err := s.db.Exec(`UPDATE oauth_states SET used = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("marking state used: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("state not found: %s", id)
	}
	return nil
}

// DeleteExpired removes expired state entries.
func (s *SQLiteStateStore) DeleteExpired() (int64, error) {
	result, err := s.db.Exec(`DELETE FROM oauth_states WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("deleting expired states: %w", err)
	}
	return result.RowsAffected()
}

// Close is a no-op; the caller manages the DB lifecycle.
func (s *SQLiteStateStore) Close() error {
	return nil
}

func generateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// generateStateToken creates a cryptographically random state token.
func generateStateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
