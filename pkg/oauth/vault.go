package oauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters for key derivation (matching pkg/auth).
const (
	vaultArgonTime    = 1
	vaultArgonMemory  = 64 * 1024 // 64 MB
	vaultArgonThreads = 4
	vaultArgonKeyLen  = 32 // AES-256
	vaultSaltLen      = 16
)

// Credential status constants.
const (
	CredentialStatusActive  = "active"
	CredentialStatusRevoked = "revoked"
	CredentialStatusExpired = "expired"
)

// ValidCredentialStatus checks if a status string is valid.
func ValidCredentialStatus(s string) bool {
	switch s {
	case CredentialStatusActive, CredentialStatusRevoked, CredentialStatusExpired:
		return true
	}
	return false
}

// VaultCredential represents a stored OAuth credential for a user+provider pair.
type VaultCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProviderID   string    `json:"provider_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scopes       string    `json:"scopes,omitempty"`
	Label        string    `json:"label,omitempty"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsExpired returns true if the access token is past its expiry time.
func (c *VaultCredential) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(c.ExpiresAt)
}

// NeedsRefresh returns true if the token expires within 5 minutes.
func (c *VaultCredential) NeedsRefresh() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().Add(5 * time.Minute).After(c.ExpiresAt)
}

// vaultTokenData is the encrypted payload stored in the database.
type vaultTokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// VaultStore abstracts per-user per-integration encrypted credential persistence.
type VaultStore interface {
	// Store saves (or updates) a credential for the given user+provider pair.
	Store(cred *VaultCredential) error
	// Get returns a credential by user and provider, or nil if not found.
	Get(userID, providerID string) (*VaultCredential, error)
	// GetByID returns a credential by its unique ID, or nil if not found.
	GetByID(id string) (*VaultCredential, error)
	// ListByUser returns all credentials for a user, optionally filtered by status.
	ListByUser(userID string, status string) ([]*VaultCredential, error)
	// Delete removes a credential by user and provider.
	Delete(userID, providerID string) error
	// DeleteByID removes a credential by its unique ID.
	DeleteByID(id string) error
	// Revoke marks a credential as revoked.
	Revoke(userID, providerID string) error
	// DeleteExpired removes credentials with expired tokens. Returns count deleted.
	DeleteExpired() (int64, error)
	// Close releases resources.
	Close() error
}

// SQLiteVaultStore implements VaultStore backed by SQLite with AES-256-GCM encryption.
type SQLiteVaultStore struct {
	db         *sql.DB
	passphrase string
	mu         sync.RWMutex
}

// NewSQLiteVaultStore creates a new SQLite-backed vault store.
// If encryptionKey is empty, tokens are stored in base64 (a warning is logged).
func NewSQLiteVaultStore(db *sql.DB, encryptionKey string) (*SQLiteVaultStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	if encryptionKey == "" {
		log.Println("WARNING: Credential vault encryption key not set. Tokens stored without encryption.")
	}
	return &SQLiteVaultStore{
		db:         db,
		passphrase: encryptionKey,
	}, nil
}

// Store saves or updates a credential. Tokens are encrypted at rest.
func (s *SQLiteVaultStore) Store(cred *VaultCredential) error {
	if cred == nil {
		return fmt.Errorf("credential is required")
	}
	if cred.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if cred.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if cred.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if cred.ID == "" {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generating ID: %w", err)
		}
		cred.ID = id
	}
	if cred.Status == "" {
		cred.Status = CredentialStatusActive
	}
	if !ValidCredentialStatus(cred.Status) {
		return fmt.Errorf("invalid status: %s", cred.Status)
	}

	// Encrypt token data.
	tokenData := vaultTokenData{
		AccessToken:  cred.AccessToken,
		RefreshToken: cred.RefreshToken,
		TokenType:    cred.TokenType,
		IDToken:      cred.IDToken,
	}
	plaintext, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("marshal token data: %w", err)
	}
	encrypted, isEncrypted, err := s.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt token data: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	encFlag := 0
	if isEncrypted {
		encFlag = 1
	}
	expiresAt := ""
	if !cred.ExpiresAt.IsZero() {
		expiresAt = cred.ExpiresAt.Format(time.RFC3339Nano)
	}

	_, err = s.db.Exec(`
		INSERT INTO credential_vault (id, user_id, provider_id, encrypted_data, encrypted, label, status, scopes, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, provider_id) DO UPDATE SET
			encrypted_data = excluded.encrypted_data,
			encrypted = excluded.encrypted,
			label = excluded.label,
			status = excluded.status,
			scopes = excluded.scopes,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at`,
		cred.ID, cred.UserID, cred.ProviderID, encrypted, encFlag,
		cred.Label, cred.Status, cred.Scopes, expiresAt, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}

	cred.UpdatedAt, _ = time.Parse(time.RFC3339Nano, now)
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = cred.UpdatedAt
	}

	return nil
}

// Get returns a credential by user and provider, or nil if not found.
func (s *SQLiteVaultStore) Get(userID, providerID string) (*VaultCredential, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if providerID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.scanCredential(
		`SELECT id, user_id, provider_id, encrypted_data, encrypted, label, status, scopes, expires_at, created_at, updated_at
		 FROM credential_vault WHERE user_id = ? AND provider_id = ?`,
		userID, providerID,
	)
}

// GetByID returns a credential by its unique ID, or nil if not found.
func (s *SQLiteVaultStore) GetByID(id string) (*VaultCredential, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.scanCredential(
		`SELECT id, user_id, provider_id, encrypted_data, encrypted, label, status, scopes, expires_at, created_at, updated_at
		 FROM credential_vault WHERE id = ?`,
		id,
	)
}

// ListByUser returns all credentials for a user, optionally filtered by status.
func (s *SQLiteVaultStore) ListByUser(userID string, status string) ([]*VaultCredential, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []interface{}
	if status != "" {
		if !ValidCredentialStatus(status) {
			return nil, fmt.Errorf("invalid status filter: %s", status)
		}
		query = `SELECT id, user_id, provider_id, encrypted_data, encrypted, label, status, scopes, expires_at, created_at, updated_at
				 FROM credential_vault WHERE user_id = ? AND status = ? ORDER BY created_at ASC`
		args = []interface{}{userID, status}
	} else {
		query = `SELECT id, user_id, provider_id, encrypted_data, encrypted, label, status, scopes, expires_at, created_at, updated_at
				 FROM credential_vault WHERE user_id = ? ORDER BY created_at ASC`
		args = []interface{}{userID}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query credentials: %w", err)
	}
	defer rows.Close()

	var creds []*VaultCredential
	for rows.Next() {
		cred, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}
	return creds, rows.Err()
}

// Delete removes a credential by user and provider.
func (s *SQLiteVaultStore) Delete(userID, providerID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if providerID == "" {
		return fmt.Errorf("provider ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM credential_vault WHERE user_id = ? AND provider_id = ?`, userID, providerID)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// DeleteByID removes a credential by its unique ID.
func (s *SQLiteVaultStore) DeleteByID(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM credential_vault WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// Revoke marks a credential as revoked.
func (s *SQLiteVaultStore) Revoke(userID, providerID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if providerID == "" {
		return fmt.Errorf("provider ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.Exec(
		`UPDATE credential_vault SET status = ?, updated_at = ? WHERE user_id = ? AND provider_id = ?`,
		CredentialStatusRevoked, now, userID, providerID,
	)
	if err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// DeleteExpired removes credentials whose tokens have expired.
func (s *SQLiteVaultStore) DeleteExpired() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.Exec(
		`DELETE FROM credential_vault WHERE expires_at != '' AND expires_at < ?`, now,
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired credentials: %w", err)
	}
	return result.RowsAffected()
}

// Close is a no-op; the caller manages the DB lifecycle.
func (s *SQLiteVaultStore) Close() error {
	return nil
}

// scanCredential executes a single-row query and decrypts the result.
func (s *SQLiteVaultStore) scanCredential(query string, args ...interface{}) (*VaultCredential, error) {
	var cred VaultCredential
	var data []byte
	var encFlag int
	var expiresAt, createdAt, updatedAt string

	err := s.db.QueryRow(query, args...).Scan(
		&cred.ID, &cred.UserID, &cred.ProviderID, &data, &encFlag,
		&cred.Label, &cred.Status, &cred.Scopes,
		&expiresAt, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query credential: %w", err)
	}

	if err := s.decryptInto(&cred, data, encFlag == 1); err != nil {
		return nil, err
	}

	cred.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	cred.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	cred.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &cred, nil
}

// scanRow scans a row from rows.Next() and decrypts the result.
func (s *SQLiteVaultStore) scanRow(rows *sql.Rows) (*VaultCredential, error) {
	var cred VaultCredential
	var data []byte
	var encFlag int
	var expiresAt, createdAt, updatedAt string

	if err := rows.Scan(
		&cred.ID, &cred.UserID, &cred.ProviderID, &data, &encFlag,
		&cred.Label, &cred.Status, &cred.Scopes,
		&expiresAt, &createdAt, &updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan credential row: %w", err)
	}

	if err := s.decryptInto(&cred, data, encFlag == 1); err != nil {
		return nil, err
	}

	cred.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	cred.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	cred.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &cred, nil
}

// decryptInto decrypts token data and populates the credential fields.
func (s *SQLiteVaultStore) decryptInto(cred *VaultCredential, data []byte, encrypted bool) error {
	plaintext, err := s.decrypt(data, encrypted)
	if err != nil {
		return fmt.Errorf("decrypt credential %q: %w", cred.ID, err)
	}

	var td vaultTokenData
	if err := json.Unmarshal(plaintext, &td); err != nil {
		return fmt.Errorf("unmarshal credential %q: %w", cred.ID, err)
	}

	cred.AccessToken = td.AccessToken
	cred.RefreshToken = td.RefreshToken
	cred.TokenType = td.TokenType
	cred.IDToken = td.IDToken
	return nil
}

// encrypt encrypts plaintext if a passphrase is set.
func (s *SQLiteVaultStore) encrypt(plaintext []byte) ([]byte, bool, error) {
	if s.passphrase == "" {
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(plaintext)))
		base64.StdEncoding.Encode(encoded, plaintext)
		return encoded, false, nil
	}

	encrypted, err := vaultEncryptAESGCM(plaintext, s.passphrase)
	if err != nil {
		return nil, false, err
	}
	return encrypted, true, nil
}

// decrypt decrypts data based on the encrypted flag.
func (s *SQLiteVaultStore) decrypt(data []byte, encrypted bool) ([]byte, error) {
	if !encrypted {
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(decoded, data)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
		return decoded[:n], nil
	}

	if s.passphrase == "" {
		return nil, fmt.Errorf("credential is encrypted but no encryption key configured")
	}

	return vaultDecryptAESGCM(data, s.passphrase)
}

// vaultDeriveKey derives a 256-bit key from a passphrase and salt using Argon2id.
func vaultDeriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, vaultArgonTime, vaultArgonMemory, vaultArgonThreads, vaultArgonKeyLen)
}

// vaultEncryptAESGCM encrypts plaintext using AES-256-GCM.
// Returns: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func vaultEncryptAESGCM(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, vaultSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	key := vaultDeriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, vaultSaltLen+len(nonce)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// vaultDecryptAESGCM decrypts data produced by vaultEncryptAESGCM.
func vaultDecryptAESGCM(data []byte, passphrase string) ([]byte, error) {
	if len(data) < vaultSaltLen+12 {
		return nil, fmt.Errorf("ciphertext too short")
	}

	salt := data[:vaultSaltLen]
	key := vaultDeriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < vaultSaltLen+nonceSize {
		return nil, fmt.Errorf("ciphertext too short for nonce")
	}

	nonce := data[vaultSaltLen : vaultSaltLen+nonceSize]
	ciphertext := data[vaultSaltLen+nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// StoreFromTokenResponse is a convenience that converts an OAuth TokenResponse
// into a VaultCredential and stores it.
func StoreFromTokenResponse(store VaultStore, resp *TokenResponse, label string) error {
	if store == nil {
		return fmt.Errorf("vault store is required")
	}
	if resp == nil {
		return fmt.Errorf("token response is required")
	}
	if resp.UserID == "" {
		return fmt.Errorf("user ID is required in token response")
	}
	if resp.ProviderID == "" {
		return fmt.Errorf("provider ID is required in token response")
	}

	cred := &VaultCredential{
		UserID:       resp.UserID,
		ProviderID:   resp.ProviderID,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		IDToken:      resp.IDToken,
		Scopes:       resp.Scope,
		Label:        label,
		Status:       CredentialStatusActive,
		ExpiresAt:    resp.ExpiresAt,
	}
	return store.Store(cred)
}

// vaultGenerateID creates a cryptographically random ID.
func vaultGenerateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
