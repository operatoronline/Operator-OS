package oauth

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func openVaultTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS credential_vault (
			id              TEXT PRIMARY KEY,
			user_id         TEXT NOT NULL,
			provider_id     TEXT NOT NULL,
			encrypted_data  BLOB NOT NULL,
			encrypted       INTEGER NOT NULL DEFAULT 1,
			label           TEXT NOT NULL DEFAULT '',
			status          TEXT NOT NULL DEFAULT 'active',
			scopes          TEXT NOT NULL DEFAULT '',
			expires_at      TEXT NOT NULL DEFAULT '',
			created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			UNIQUE(user_id, provider_id)
		)`)
	require.NoError(t, err)
	return db
}

func newTestVault(t *testing.T, key string) (*SQLiteVaultStore, *sql.DB) {
	t.Helper()
	db := openVaultTestDB(t)
	store, err := NewSQLiteVaultStore(db, key)
	require.NoError(t, err)
	return store, db
}

// --- Constructor ---

func TestNewSQLiteVaultStore_NilDB(t *testing.T) {
	_, err := NewSQLiteVaultStore(nil, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

func TestNewSQLiteVaultStore_OK(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	assert.NotNil(t, store)
}

func TestNewSQLiteVaultStore_NoKey(t *testing.T) {
	// Should succeed with warning (base64 mode)
	store, _ := newTestVault(t, "")
	assert.NotNil(t, store)
}

// --- ValidCredentialStatus ---

func TestValidCredentialStatus(t *testing.T) {
	assert.True(t, ValidCredentialStatus("active"))
	assert.True(t, ValidCredentialStatus("revoked"))
	assert.True(t, ValidCredentialStatus("expired"))
	assert.False(t, ValidCredentialStatus("invalid"))
	assert.False(t, ValidCredentialStatus(""))
}

// --- VaultCredential methods ---

func TestVaultCredential_IsExpired(t *testing.T) {
	c := &VaultCredential{}
	assert.False(t, c.IsExpired(), "zero time = not expired")

	c.ExpiresAt = time.Now().UTC().Add(time.Hour)
	assert.False(t, c.IsExpired())

	c.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	assert.True(t, c.IsExpired())
}

func TestVaultCredential_NeedsRefresh(t *testing.T) {
	c := &VaultCredential{}
	assert.False(t, c.NeedsRefresh(), "zero time = no refresh")

	c.ExpiresAt = time.Now().UTC().Add(time.Hour)
	assert.False(t, c.NeedsRefresh())

	c.ExpiresAt = time.Now().UTC().Add(3 * time.Minute)
	assert.True(t, c.NeedsRefresh())
}

// --- Store ---

func TestVault_Store_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	cred := &VaultCredential{
		UserID:       "user1",
		ProviderID:   "google",
		AccessToken:  "at-123",
		RefreshToken: "rt-456",
		TokenType:    "Bearer",
		Scopes:       "email profile",
		Label:        "My Google",
	}
	err := store.Store(cred)
	require.NoError(t, err)
	assert.NotEmpty(t, cred.ID)
	assert.Equal(t, CredentialStatusActive, cred.Status)
}

func TestVault_Store_NilCredential(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Store(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credential is required")
}

func TestVault_Store_EmptyUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Store(&VaultCredential{ProviderID: "google", AccessToken: "at"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID")
}

func TestVault_Store_EmptyProviderID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Store(&VaultCredential{UserID: "u1", AccessToken: "at"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider ID")
}

func TestVault_Store_EmptyAccessToken(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Store(&VaultCredential{UserID: "u1", ProviderID: "google"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access token")
}

func TestVault_Store_InvalidStatus(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at", Status: "bad",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestVault_Store_Upsert(t *testing.T) {
	store, _ := newTestVault(t, "test-key")

	cred1 := &VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at-1",
	}
	require.NoError(t, store.Store(cred1))

	// Update same user+provider
	cred2 := &VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at-2", RefreshToken: "rt-new",
	}
	require.NoError(t, store.Store(cred2))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, "at-2", got.AccessToken)
	assert.Equal(t, "rt-new", got.RefreshToken)
}

func TestVault_Store_WithExpiry(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	exp := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)
	cred := &VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at",
		ExpiresAt: exp,
	}
	require.NoError(t, store.Store(cred))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	// Times should be very close (within a second due to formatting)
	assert.WithinDuration(t, exp, got.ExpiresAt, time.Second)
}

// --- Get ---

func TestVault_Get_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	cred := &VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at-123",
		RefreshToken: "rt-456", TokenType: "Bearer", IDToken: "id-789",
		Scopes: "email", Label: "work",
	}
	require.NoError(t, store.Store(cred))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "at-123", got.AccessToken)
	assert.Equal(t, "rt-456", got.RefreshToken)
	assert.Equal(t, "Bearer", got.TokenType)
	assert.Equal(t, "id-789", got.IDToken)
	assert.Equal(t, "email", got.Scopes)
	assert.Equal(t, "work", got.Label)
	assert.Equal(t, CredentialStatusActive, got.Status)
}

func TestVault_Get_NotFound(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVault_Get_EmptyUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	_, err := store.Get("", "google")
	assert.Error(t, err)
}

func TestVault_Get_EmptyProviderID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	_, err := store.Get("u1", "")
	assert.Error(t, err)
}

// --- GetByID ---

func TestVault_GetByID_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	cred := &VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "at-123",
	}
	require.NoError(t, store.Store(cred))

	got, err := store.GetByID(cred.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "at-123", got.AccessToken)
}

func TestVault_GetByID_NotFound(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	got, err := store.GetByID("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVault_GetByID_EmptyID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	_, err := store.GetByID("")
	assert.Error(t, err)
}

// --- ListByUser ---

func TestVault_ListByUser_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at1"}))
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "github", AccessToken: "at2"}))

	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestVault_ListByUser_StatusFilter(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at1"}))
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "github", AccessToken: "at2"}))
	require.NoError(t, store.Revoke("u1", "github"))

	list, err := store.ListByUser("u1", CredentialStatusActive)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "google", list[0].ProviderID)
}

func TestVault_ListByUser_InvalidStatus(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	_, err := store.ListByUser("u1", "bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestVault_ListByUser_Empty(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestVault_ListByUser_EmptyUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	_, err := store.ListByUser("", "")
	assert.Error(t, err)
}

// --- Delete ---

func TestVault_Delete_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at"}))

	err := store.Delete("u1", "google")
	require.NoError(t, err)

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVault_Delete_NotFound(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Delete("u1", "google")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVault_Delete_EmptyUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Delete("", "google")
	assert.Error(t, err)
}

func TestVault_Delete_EmptyProviderID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Delete("u1", "")
	assert.Error(t, err)
}

// --- DeleteByID ---

func TestVault_DeleteByID_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	cred := &VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at"}
	require.NoError(t, store.Store(cred))

	err := store.DeleteByID(cred.ID)
	require.NoError(t, err)

	got, err := store.GetByID(cred.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestVault_DeleteByID_NotFound(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.DeleteByID("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVault_DeleteByID_EmptyID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.DeleteByID("")
	assert.Error(t, err)
}

// --- Revoke ---

func TestVault_Revoke_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at"}))

	err := store.Revoke("u1", "google")
	require.NoError(t, err)

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, CredentialStatusRevoked, got.Status)
}

func TestVault_Revoke_NotFound(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Revoke("u1", "google")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVault_Revoke_EmptyUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Revoke("", "google")
	assert.Error(t, err)
}

func TestVault_Revoke_EmptyProviderID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Revoke("u1", "")
	assert.Error(t, err)
}

// --- DeleteExpired ---

func TestVault_DeleteExpired_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")

	// Expired credential
	require.NoError(t, store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "old", AccessToken: "at",
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	}))
	// Active credential
	require.NoError(t, store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "new", AccessToken: "at",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}))
	// No expiry (should not be deleted)
	require.NoError(t, store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "forever", AccessToken: "at",
	}))

	n, err := store.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestVault_DeleteExpired_None(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	n, err := store.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

// --- Multi-user isolation ---

func TestVault_MultiUserIsolation(t *testing.T) {
	store, _ := newTestVault(t, "test-key")

	require.NoError(t, store.Store(&VaultCredential{UserID: "u1", ProviderID: "google", AccessToken: "at-u1"}))
	require.NoError(t, store.Store(&VaultCredential{UserID: "u2", ProviderID: "google", AccessToken: "at-u2"}))

	// User1 should see only their credential
	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "at-u1", list[0].AccessToken)

	// User2 should see only their credential
	list, err = store.ListByUser("u2", "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "at-u2", list[0].AccessToken)

	// Deleting user1's credential should not affect user2
	require.NoError(t, store.Delete("u1", "google"))
	got, err := store.Get("u2", "google")
	require.NoError(t, err)
	assert.NotNil(t, got)
}

// --- Encryption modes ---

func TestVault_EncryptedMode(t *testing.T) {
	store, _ := newTestVault(t, "my-secret-key")

	require.NoError(t, store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "secret-token",
	}))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, "secret-token", got.AccessToken)
}

func TestVault_UnencryptedMode(t *testing.T) {
	store, _ := newTestVault(t, "")

	require.NoError(t, store.Store(&VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "plain-token",
	}))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, "plain-token", got.AccessToken)
}

func TestVault_EncryptionKeyMismatch(t *testing.T) {
	db := openVaultTestDB(t)

	// Store with key A
	storeA, err := NewSQLiteVaultStore(db, "key-A")
	require.NoError(t, err)
	require.NoError(t, storeA.Store(&VaultCredential{
		UserID: "u1", ProviderID: "google", AccessToken: "secret",
	}))

	// Try to read with key B
	storeB, err := NewSQLiteVaultStore(db, "key-B")
	require.NoError(t, err)
	_, err = storeB.Get("u1", "google")
	assert.Error(t, err) // decryption should fail
}

// --- StoreFromTokenResponse ---

func TestStoreFromTokenResponse_Success(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	resp := &TokenResponse{
		AccessToken:  "at-123",
		RefreshToken: "rt-456",
		TokenType:    "Bearer",
		IDToken:      "id-789",
		Scope:        "email",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
		ProviderID:   "google",
		UserID:       "u1",
	}

	err := StoreFromTokenResponse(store, resp, "My Google")
	require.NoError(t, err)

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "at-123", got.AccessToken)
	assert.Equal(t, "rt-456", got.RefreshToken)
	assert.Equal(t, "Bearer", got.TokenType)
	assert.Equal(t, "id-789", got.IDToken)
	assert.Equal(t, "email", got.Scopes)
	assert.Equal(t, "My Google", got.Label)
}

func TestStoreFromTokenResponse_NilStore(t *testing.T) {
	err := StoreFromTokenResponse(nil, &TokenResponse{}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vault store is required")
}

func TestStoreFromTokenResponse_NilResponse(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := StoreFromTokenResponse(store, nil, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token response is required")
}

func TestStoreFromTokenResponse_MissingUserID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := StoreFromTokenResponse(store, &TokenResponse{ProviderID: "google", AccessToken: "at"}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID")
}

func TestStoreFromTokenResponse_MissingProviderID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := StoreFromTokenResponse(store, &TokenResponse{UserID: "u1", AccessToken: "at"}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider ID")
}

// --- Encryption helpers ---

func TestVaultEncryptDecrypt(t *testing.T) {
	plaintext := []byte("hello world, this is secret data!")
	passphrase := "strong-passphrase-123"

	encrypted, err := vaultEncryptAESGCM(plaintext, passphrase)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := vaultDecryptAESGCM(encrypted, passphrase)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestVaultDecrypt_TooShort(t *testing.T) {
	_, err := vaultDecryptAESGCM([]byte("short"), "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestVaultDecrypt_WrongKey(t *testing.T) {
	encrypted, err := vaultEncryptAESGCM([]byte("secret"), "key1")
	require.NoError(t, err)

	_, err = vaultDecryptAESGCM(encrypted, "key2")
	assert.Error(t, err)
}

// --- Close ---

func TestVault_Close(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	err := store.Close()
	assert.NoError(t, err)
}

// --- Custom ID ---

func TestVault_Store_CustomID(t *testing.T) {
	store, _ := newTestVault(t, "test-key")
	cred := &VaultCredential{
		ID:          "custom-id-123",
		UserID:      "u1",
		ProviderID:  "google",
		AccessToken: "at",
	}
	require.NoError(t, store.Store(cred))
	assert.Equal(t, "custom-id-123", cred.ID)

	got, err := store.GetByID("custom-id-123")
	require.NoError(t, err)
	require.NotNil(t, got)
}

// --- Full flow ---

func TestVault_FullFlow(t *testing.T) {
	store, _ := newTestVault(t, "production-key-256")

	// 1. Store credential after OAuth callback
	cred := &VaultCredential{
		UserID:       "user-abc",
		ProviderID:   "google",
		AccessToken:  "ya29.access-token",
		RefreshToken: "1//refresh-token",
		TokenType:    "Bearer",
		IDToken:      "eyJ...",
		Scopes:       "email profile calendar",
		Label:        "Work Gmail",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}
	require.NoError(t, store.Store(cred))

	// 2. Retrieve it
	got, err := store.Get("user-abc", "google")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ya29.access-token", got.AccessToken)
	assert.Equal(t, "Work Gmail", got.Label)
	assert.False(t, got.NeedsRefresh())

	// 3. Update after token refresh
	cred.AccessToken = "ya29.new-access-token"
	cred.ExpiresAt = time.Now().UTC().Add(time.Hour)
	require.NoError(t, store.Store(cred))

	got, err = store.Get("user-abc", "google")
	require.NoError(t, err)
	assert.Equal(t, "ya29.new-access-token", got.AccessToken)

	// 4. Add another integration
	require.NoError(t, store.Store(&VaultCredential{
		UserID: "user-abc", ProviderID: "github", AccessToken: "gho_token",
		Label: "GitHub",
	}))

	// 5. List all
	list, err := store.ListByUser("user-abc", "")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// 6. Revoke one
	require.NoError(t, store.Revoke("user-abc", "github"))
	list, err = store.ListByUser("user-abc", CredentialStatusActive)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 7. Delete
	require.NoError(t, store.Delete("user-abc", "google"))
	list, err = store.ListByUser("user-abc", "")
	require.NoError(t, err)
	assert.Len(t, list, 1) // only revoked github remains
}
