package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/standardws/operator/pkg/fileutil"
)

// globalStore is the optional CredentialStore backend. When set, the
// package-level functions (GetCredential, SetCredential, etc.) delegate to it
// instead of the legacy JSON file.
var (
	globalStore   CredentialStore
	globalStoreMu sync.RWMutex
)

// SetGlobalCredentialStore sets the global CredentialStore used by the
// package-level convenience functions. Pass nil to revert to JSON file mode.
func SetGlobalCredentialStore(store CredentialStore) {
	globalStoreMu.Lock()
	defer globalStoreMu.Unlock()
	globalStore = store
}

// GetGlobalCredentialStore returns the current global CredentialStore, or nil
// if none is set (legacy JSON mode).
func GetGlobalCredentialStore() CredentialStore {
	globalStoreMu.RLock()
	defer globalStoreMu.RUnlock()
	return globalStore
}

type AuthCredential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Provider     string    `json:"provider"`
	AuthMethod   string    `json:"auth_method"`
	Email        string    `json:"email,omitempty"`
	ProjectID    string    `json:"project_id,omitempty"`
}

type AuthStore struct {
	Credentials map[string]*AuthCredential `json:"credentials"`
}

func (c *AuthCredential) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

func (c *AuthCredential) NeedsRefresh() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

func authFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".operator", "auth.json")
}

func LoadStore() (*AuthStore, error) {
	if s := GetGlobalCredentialStore(); s != nil {
		creds, err := s.List()
		if err != nil {
			return nil, err
		}
		return &AuthStore{Credentials: creds}, nil
	}

	path := authFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthStore{Credentials: make(map[string]*AuthCredential)}, nil
		}
		return nil, err
	}

	var store AuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Credentials == nil {
		store.Credentials = make(map[string]*AuthCredential)
	}
	return &store, nil
}

func SaveStore(store *AuthStore) error {
	path := authFilePath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func GetCredential(provider string) (*AuthCredential, error) {
	if s := GetGlobalCredentialStore(); s != nil {
		return s.Get(provider)
	}

	store, err := LoadStore()
	if err != nil {
		return nil, err
	}
	cred, ok := store.Credentials[provider]
	if !ok {
		return nil, nil
	}
	return cred, nil
}

func SetCredential(provider string, cred *AuthCredential) error {
	if s := GetGlobalCredentialStore(); s != nil {
		return s.Set(provider, cred)
	}

	store, err := LoadStore()
	if err != nil {
		return err
	}
	store.Credentials[provider] = cred
	return SaveStore(store)
}

func DeleteCredential(provider string) error {
	if s := GetGlobalCredentialStore(); s != nil {
		return s.Delete(provider)
	}

	store, err := LoadStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, provider)
	return SaveStore(store)
}

func DeleteAllCredentials() error {
	if s := GetGlobalCredentialStore(); s != nil {
		return s.DeleteAll()
	}

	path := authFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
