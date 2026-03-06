package auth

import (
	"path/filepath"
	"testing"
	"time"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test_credentials.db")
}

func TestSQLiteCredentialStore_NewStore(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()
}

func TestSQLiteCredentialStore_SetAndGet(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	cred := &AuthCredential{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		AccountID:    "acct-789",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		Provider:     "openai",
		AuthMethod:   "oauth",
		Email:        "test@example.com",
	}

	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	loaded, err := store.Get("openai")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("Get() returned nil")
	}
	if loaded.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, cred.AccessToken)
	}
	if loaded.RefreshToken != cred.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, cred.RefreshToken)
	}
	if loaded.AccountID != cred.AccountID {
		t.Errorf("AccountID = %q, want %q", loaded.AccountID, cred.AccountID)
	}
	if loaded.Provider != cred.Provider {
		t.Errorf("Provider = %q, want %q", loaded.Provider, cred.Provider)
	}
	if loaded.AuthMethod != cred.AuthMethod {
		t.Errorf("AuthMethod = %q, want %q", loaded.AuthMethod, cred.AuthMethod)
	}
	if loaded.Email != cred.Email {
		t.Errorf("Email = %q, want %q", loaded.Email, cred.Email)
	}
}

func TestSQLiteCredentialStore_GetNotFound(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	loaded, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent provider")
	}
}

func TestSQLiteCredentialStore_Update(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	cred := &AuthCredential{AccessToken: "old-token", Provider: "openai", AuthMethod: "oauth"}
	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	cred.AccessToken = "new-token"
	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() update error: %v", err)
	}

	loaded, err := store.Get("openai")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded.AccessToken != "new-token" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "new-token")
	}
}

func TestSQLiteCredentialStore_Delete(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	cred := &AuthCredential{AccessToken: "to-delete", Provider: "openai", AuthMethod: "oauth"}
	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	if err := store.Delete("openai"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	loaded, err := store.Get("openai")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestSQLiteCredentialStore_DeleteNonexistent(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	// Should not error when deleting a provider that doesn't exist.
	if err := store.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestSQLiteCredentialStore_DeleteAll(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	store.Set("openai", &AuthCredential{AccessToken: "a", Provider: "openai", AuthMethod: "oauth"})
	store.Set("anthropic", &AuthCredential{AccessToken: "b", Provider: "anthropic", AuthMethod: "token"})

	if err := store.DeleteAll(); err != nil {
		t.Fatalf("DeleteAll() error: %v", err)
	}

	creds, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials after DeleteAll, got %d", len(creds))
	}
}

func TestSQLiteCredentialStore_List(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	store.Set("openai", &AuthCredential{AccessToken: "a", Provider: "openai", AuthMethod: "oauth"})
	store.Set("anthropic", &AuthCredential{AccessToken: "b", Provider: "anthropic", AuthMethod: "token"})
	store.Set("google", &AuthCredential{AccessToken: "c", Provider: "google", AuthMethod: "oauth"})

	creds, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(creds) != 3 {
		t.Fatalf("expected 3 credentials, got %d", len(creds))
	}
	if creds["openai"].AccessToken != "a" {
		t.Errorf("openai token = %q, want %q", creds["openai"].AccessToken, "a")
	}
	if creds["anthropic"].AccessToken != "b" {
		t.Errorf("anthropic token = %q, want %q", creds["anthropic"].AccessToken, "b")
	}
	if creds["google"].AccessToken != "c" {
		t.Errorf("google token = %q, want %q", creds["google"].AccessToken, "c")
	}
}

func TestSQLiteCredentialStore_ListEmpty(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	creds, err := store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials, got %d", len(creds))
	}
}

func TestSQLiteCredentialStore_NoEncryptionKey(t *testing.T) {
	// With empty encryption key, should still work (base64 mode).
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	cred := &AuthCredential{
		AccessToken: "unencrypted-token",
		Provider:    "openai",
		AuthMethod:  "token",
	}
	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	loaded, err := store.Get("openai")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("Get() returned nil")
	}
	if loaded.AccessToken != "unencrypted-token" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "unencrypted-token")
	}
}

func TestSQLiteCredentialStore_DifferentKeysCannotDecrypt(t *testing.T) {
	dbPath := tempDBPath(t)

	// Store with key "alpha".
	store1, err := NewSQLiteCredentialStore(dbPath, "alpha")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	cred := &AuthCredential{AccessToken: "secret", Provider: "openai", AuthMethod: "token"}
	if err := store1.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	store1.Close()

	// Try to read with key "beta".
	store2, err := NewSQLiteCredentialStore(dbPath, "beta")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store2.Close()

	_, err = store2.Get("openai")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key, got nil")
	}
}

func TestSQLiteCredentialStore_EncryptionRoundtrip(t *testing.T) {
	dbPath := tempDBPath(t)

	store, err := NewSQLiteCredentialStore(dbPath, "strong-passphrase-123!")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}

	cred := &AuthCredential{
		AccessToken:  "super-secret-token",
		RefreshToken: "super-secret-refresh",
		AccountID:    "acct-999",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Truncate(time.Second),
		Provider:     "openai",
		AuthMethod:   "oauth",
		Email:        "user@example.com",
		ProjectID:    "proj-123",
	}

	if err := store.Set("openai", cred); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	store.Close()

	// Reopen with same key.
	store2, err := NewSQLiteCredentialStore(dbPath, "strong-passphrase-123!")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() reopen error: %v", err)
	}
	defer store2.Close()

	loaded, err := store2.Get("openai")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("Get() returned nil")
	}
	if loaded.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, cred.AccessToken)
	}
	if loaded.RefreshToken != cred.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, cred.RefreshToken)
	}
	if loaded.ProjectID != cred.ProjectID {
		t.Errorf("ProjectID = %q, want %q", loaded.ProjectID, cred.ProjectID)
	}
}

func TestSQLiteCredentialStore_GlobalStoreIntegration(t *testing.T) {
	// Test that package-level functions delegate to the global store.
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	// Set global store.
	SetGlobalCredentialStore(store)
	defer SetGlobalCredentialStore(nil)

	cred := &AuthCredential{
		AccessToken: "global-test-token",
		Provider:    "anthropic",
		AuthMethod:  "token",
	}

	// Use package-level functions.
	if err := SetCredential("anthropic", cred); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	loaded, err := GetCredential("anthropic")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("GetCredential() returned nil")
	}
	if loaded.AccessToken != "global-test-token" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "global-test-token")
	}

	// LoadStore should also work.
	authStore, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(authStore.Credentials) != 1 {
		t.Errorf("expected 1 credential, got %d", len(authStore.Credentials))
	}

	// Delete.
	if err := DeleteCredential("anthropic"); err != nil {
		t.Fatalf("DeleteCredential() error: %v", err)
	}
	loaded, err = GetCredential("anthropic")
	if err != nil {
		t.Fatalf("GetCredential() after delete error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestSQLiteCredentialStore_GlobalStoreDeleteAll(t *testing.T) {
	store, err := NewSQLiteCredentialStore(tempDBPath(t), "test-key")
	if err != nil {
		t.Fatalf("NewSQLiteCredentialStore() error: %v", err)
	}
	defer store.Close()

	SetGlobalCredentialStore(store)
	defer SetGlobalCredentialStore(nil)

	SetCredential("a", &AuthCredential{AccessToken: "1", Provider: "a", AuthMethod: "token"})
	SetCredential("b", &AuthCredential{AccessToken: "2", Provider: "b", AuthMethod: "token"})

	if err := DeleteAllCredentials(); err != nil {
		t.Fatalf("DeleteAllCredentials() error: %v", err)
	}

	authStore, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(authStore.Credentials) != 0 {
		t.Errorf("expected 0 credentials after DeleteAll, got %d", len(authStore.Credentials))
	}
}
