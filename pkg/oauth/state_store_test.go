package oauth

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE oauth_states (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			provider_id   TEXT NOT NULL,
			state         TEXT NOT NULL UNIQUE,
			code_verifier TEXT DEFAULT '',
			redirect_uri  TEXT DEFAULT '',
			scopes        TEXT DEFAULT '',
			created_at    TEXT NOT NULL,
			expires_at    TEXT NOT NULL,
			used          INTEGER DEFAULT 0
		)`)
	require.NoError(t, err)
	return db
}

func TestNewSQLiteStateStore_NilDB(t *testing.T) {
	_, err := NewSQLiteStateStore(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

func TestNewSQLiteStateStore_OK(t *testing.T) {
	db := testDB(t)
	store, err := NewSQLiteStateStore(db)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestStateStore_Create(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	state := &OAuthState{
		UserID:       "user-1",
		ProviderID:   "google",
		State:        "test-state-token",
		CodeVerifier: "test-verifier",
		RedirectURI:  "/dashboard",
		Scopes:       "openid email",
	}
	require.NoError(t, store.Create(state))
	assert.NotEmpty(t, state.ID)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.ExpiresAt.IsZero())
}

func TestStateStore_Create_Validation(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	// Nil state.
	assert.Error(t, store.Create(nil))

	// Missing state token.
	assert.Error(t, store.Create(&OAuthState{UserID: "u1", ProviderID: "p1"}))

	// Missing user ID.
	assert.Error(t, store.Create(&OAuthState{State: "s1", ProviderID: "p1"}))

	// Missing provider ID.
	assert.Error(t, store.Create(&OAuthState{State: "s1", UserID: "u1"}))
}

func TestStateStore_GetByState(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	state := &OAuthState{
		UserID:       "user-1",
		ProviderID:   "google",
		State:        "lookup-state",
		CodeVerifier: "verifier-abc",
		Scopes:       "openid",
	}
	require.NoError(t, store.Create(state))

	// Found.
	got, err := store.GetByState("lookup-state")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "google", got.ProviderID)
	assert.Equal(t, "verifier-abc", got.CodeVerifier)
	assert.False(t, got.Used)

	// Not found.
	got, err = store.GetByState("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)

	// Empty token.
	_, err = store.GetByState("")
	require.Error(t, err)
}

func TestStateStore_MarkUsed(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	state := &OAuthState{
		UserID:     "user-1",
		ProviderID: "google",
		State:      "mark-used-state",
	}
	require.NoError(t, store.Create(state))

	// Mark used.
	require.NoError(t, store.MarkUsed(state.ID))

	// Verify.
	got, err := store.GetByState("mark-used-state")
	require.NoError(t, err)
	assert.True(t, got.Used)

	// Not found.
	err = store.MarkUsed("nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Empty ID.
	assert.Error(t, store.MarkUsed(""))
}

func TestStateStore_DeleteExpired(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	// Create an expired state.
	expired := &OAuthState{
		UserID:     "user-1",
		ProviderID: "google",
		State:      "expired-state",
		CreatedAt:  time.Now().UTC().Add(-20 * time.Minute),
		ExpiresAt:  time.Now().UTC().Add(-10 * time.Minute),
	}
	require.NoError(t, store.Create(expired))

	// Create a valid state.
	valid := &OAuthState{
		UserID:     "user-1",
		ProviderID: "google",
		State:      "valid-state",
	}
	require.NoError(t, store.Create(valid))

	// Delete expired.
	n, err := store.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	// Valid state still exists.
	got, err := store.GetByState("valid-state")
	require.NoError(t, err)
	assert.NotNil(t, got)

	// Expired state is gone.
	got, err = store.GetByState("expired-state")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStateStore_Close(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)
	assert.NoError(t, store.Close())
}

func TestStateStore_CustomTTL(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	now := time.Now().UTC()
	state := &OAuthState{
		UserID:     "user-1",
		ProviderID: "google",
		State:      "custom-ttl",
		CreatedAt:  now,
		ExpiresAt:  now.Add(30 * time.Minute), // custom 30min TTL
	}
	require.NoError(t, store.Create(state))

	got, err := store.GetByState("custom-ttl")
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(30*time.Minute), got.ExpiresAt, 2*time.Second)
}

func TestStateStore_DuplicateState(t *testing.T) {
	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)

	state1 := &OAuthState{
		UserID:     "user-1",
		ProviderID: "google",
		State:      "same-state",
	}
	require.NoError(t, store.Create(state1))

	state2 := &OAuthState{
		UserID:     "user-2",
		ProviderID: "github",
		State:      "same-state",
	}
	assert.Error(t, store.Create(state2))
}

func TestGenerateStateToken(t *testing.T) {
	token1, err := generateStateToken()
	require.NoError(t, err)
	assert.Len(t, token1, 64) // 32 bytes = 64 hex chars

	token2, err := generateStateToken()
	require.NoError(t, err)
	assert.NotEqual(t, token1, token2) // uniqueness
}

func TestGenerateID(t *testing.T) {
	id1, err := generateID()
	require.NoError(t, err)
	assert.Len(t, id1, 32) // 16 bytes = 32 hex chars

	id2, err := generateID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2)
}
