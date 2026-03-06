package auth

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteCredentialStore implements CredentialStore backed by a SQLite database
// with AES-256-GCM encryption for credential data.
//
// If no encryption key is provided, credentials are stored in base64 encoding
// (not plaintext) with a loud warning. This allows self-hosted users to run
// without configuring encryption, while making it clear that at-rest encryption
// is strongly recommended.
type SQLiteCredentialStore struct {
	db         *sql.DB
	passphrase string // empty = unencrypted (base64-only) mode
	mu         sync.RWMutex
}

// NewSQLiteCredentialStore opens (or creates) a SQLite database at dbPath and
// initialises the credentials schema. If encryptionKey is empty, credentials
// are stored in base64 without encryption (a warning is logged).
func NewSQLiteCredentialStore(dbPath string, encryptionKey string) (*SQLiteCredentialStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)

	if err := initCredentialSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init credential schema: %w", err)
	}

	if encryptionKey == "" {
		log.Println("WARNING: OPERATOR_ENCRYPTION_KEY not set. Credentials will be stored without encryption. " +
			"Set OPERATOR_ENCRYPTION_KEY environment variable for at-rest encryption.")
	}

	return &SQLiteCredentialStore{
		db:         db,
		passphrase: encryptionKey,
	}, nil
}

func initCredentialSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS credentials (
    provider        TEXT PRIMARY KEY,
    encrypted_data  BLOB NOT NULL,
    encrypted       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now'))
);
`
	_, err := db.Exec(schema)
	return err
}

// Get returns the credential for the given provider, or nil if not found.
func (s *SQLiteCredentialStore) Get(provider string) (*AuthCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []byte
	var encrypted int
	err := s.db.QueryRow(
		`SELECT encrypted_data, encrypted FROM credentials WHERE provider = ?`, provider,
	).Scan(&data, &encrypted)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query credential %q: %w", provider, err)
	}

	plaintext, err := s.decrypt(data, encrypted == 1)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential %q: %w", provider, err)
	}

	var cred AuthCredential
	if err := json.Unmarshal(plaintext, &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credential %q: %w", provider, err)
	}
	return &cred, nil
}

// Set stores (or updates) the credential for the given provider.
func (s *SQLiteCredentialStore) Set(provider string, cred *AuthCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plaintext, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}

	data, isEncrypted, err := s.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt credential: %w", err)
	}

	now := time.Now().Format(time.RFC3339Nano)
	encFlag := 0
	if isEncrypted {
		encFlag = 1
	}

	_, err = s.db.Exec(
		`INSERT INTO credentials (provider, encrypted_data, encrypted, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(provider) DO UPDATE SET
		     encrypted_data = excluded.encrypted_data,
		     encrypted = excluded.encrypted,
		     updated_at = excluded.updated_at`,
		provider, data, encFlag, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert credential %q: %w", provider, err)
	}
	return nil
}

// Delete removes the credential for the given provider.
func (s *SQLiteCredentialStore) Delete(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM credentials WHERE provider = ?`, provider)
	if err != nil {
		return fmt.Errorf("delete credential %q: %w", provider, err)
	}
	return nil
}

// DeleteAll removes all stored credentials.
func (s *SQLiteCredentialStore) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM credentials`)
	if err != nil {
		return fmt.Errorf("delete all credentials: %w", err)
	}
	return nil
}

// List returns all stored credentials keyed by provider name.
func (s *SQLiteCredentialStore) List() (map[string]*AuthCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT provider, encrypted_data, encrypted FROM credentials`)
	if err != nil {
		return nil, fmt.Errorf("query credentials: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*AuthCredential)
	for rows.Next() {
		var provider string
		var data []byte
		var encrypted int
		if err := rows.Scan(&provider, &data, &encrypted); err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}

		plaintext, err := s.decrypt(data, encrypted == 1)
		if err != nil {
			return nil, fmt.Errorf("decrypt credential %q: %w", provider, err)
		}

		var cred AuthCredential
		if err := json.Unmarshal(plaintext, &cred); err != nil {
			return nil, fmt.Errorf("unmarshal credential %q: %w", provider, err)
		}
		result[provider] = &cred
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credentials: %w", err)
	}

	return result, nil
}

// Close closes the underlying database connection.
func (s *SQLiteCredentialStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// encrypt encrypts plaintext if a passphrase is set.
// Returns (data, isEncrypted, error).
func (s *SQLiteCredentialStore) encrypt(plaintext []byte) ([]byte, bool, error) {
	if s.passphrase == "" {
		// No encryption key — store as base64 (not plaintext).
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(plaintext)))
		base64.StdEncoding.Encode(encoded, plaintext)
		return encoded, false, nil
	}

	encrypted, err := encryptAESGCM(plaintext, s.passphrase)
	if err != nil {
		return nil, false, err
	}
	return encrypted, true, nil
}

// decrypt decrypts data based on the encrypted flag.
func (s *SQLiteCredentialStore) decrypt(data []byte, encrypted bool) ([]byte, error) {
	if !encrypted {
		// Base64-encoded, not encrypted.
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(decoded, data)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
		return decoded[:n], nil
	}

	if s.passphrase == "" {
		return nil, fmt.Errorf("credential is encrypted but no encryption key is configured (set OPERATOR_ENCRYPTION_KEY)")
	}

	return decryptAESGCM(data, s.passphrase)
}
