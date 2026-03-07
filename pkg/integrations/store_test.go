package integrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE user_integrations (
			id              TEXT PRIMARY KEY,
			user_id         TEXT NOT NULL,
			integration_id  TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			config          TEXT DEFAULT '{}',
			scopes          TEXT DEFAULT '[]',
			error_message   TEXT DEFAULT '',
			last_used_at    TEXT,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL,
			UNIQUE(user_id, integration_id)
		)`)
	require.NoError(t, err)
	return db
}

// --- Constructor ---

func TestNewSQLiteUserIntegrationStore_NilDB(t *testing.T) {
	_, err := NewSQLiteUserIntegrationStore(nil)
	assert.ErrorContains(t, err, "nil")
}

func TestNewSQLiteUserIntegrationStore_OK(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, err := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

// --- Create ---

func TestStore_Create_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)

	ui := &UserIntegration{
		UserID:        "user1",
		IntegrationID: "google",
		Config:        map[string]string{"domain": "gmail.com"},
		Scopes:        []string{"read", "write"},
	}
	err := store.Create(ui)
	require.NoError(t, err)
	assert.NotEmpty(t, ui.ID)
	assert.Equal(t, UserIntegrationPending, ui.Status)
	assert.False(t, ui.CreatedAt.IsZero())
}

func TestStore_Create_Nil(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Create(nil), "nil")
}

func TestStore_Create_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Create(&UserIntegration{IntegrationID: "g"}), "user_id is required")
}

func TestStore_Create_EmptyIntegrationID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Create(&UserIntegration{UserID: "u1"}), "integration_id is required")
}

func TestStore_Create_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Create(&UserIntegration{
		UserID: "u1", IntegrationID: "g", Status: "bad",
	}), "invalid status")
}

func TestStore_Create_Duplicate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{UserID: "u1", IntegrationID: "google"}
	require.NoError(t, store.Create(ui))
	err := store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"})
	assert.ErrorContains(t, err, "already connected")
}

func TestStore_Create_CustomID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{ID: "custom-id", UserID: "u1", IntegrationID: "google"}
	require.NoError(t, store.Create(ui))
	assert.Equal(t, "custom-id", ui.ID)
}

func TestStore_Create_WithScopes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{
		UserID:        "u1",
		IntegrationID: "google",
		Scopes:        []string{"email", "calendar"},
	}
	require.NoError(t, store.Create(ui))
	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, []string{"email", "calendar"}, got.Scopes)
}

// --- Get ---

func TestStore_Get_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{UserID: "u1", IntegrationID: "google", Config: map[string]string{"k": "v"}}
	require.NoError(t, store.Create(ui))

	got, err := store.Get("u1", "google")
	require.NoError(t, err)
	assert.Equal(t, "u1", got.UserID)
	assert.Equal(t, "google", got.IntegrationID)
	assert.Equal(t, "v", got.Config["k"])
}

func TestStore_Get_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.Get("u1", "nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_Get_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.Get("", "google")
	assert.ErrorContains(t, err, "user_id is required")
}

func TestStore_Get_EmptyIntegrationID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.Get("u1", "")
	assert.ErrorContains(t, err, "integration_id is required")
}

// --- GetByID ---

func TestStore_GetByID_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{UserID: "u1", IntegrationID: "google"}
	require.NoError(t, store.Create(ui))

	got, err := store.GetByID(ui.ID)
	require.NoError(t, err)
	assert.Equal(t, ui.ID, got.ID)
}

func TestStore_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.GetByID("nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_GetByID_Empty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.GetByID("")
	assert.ErrorContains(t, err, "id is required")
}

// --- ListByUser ---

func TestStore_ListByUser_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "shopify"}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u2", IntegrationID: "google"}))

	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestStore_ListByUser_StatusFilter(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google", Status: UserIntegrationActive}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "shopify", Status: UserIntegrationPending}))

	list, err := store.ListByUser("u1", UserIntegrationActive)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "google", list[0].IntegrationID)
}

func TestStore_ListByUser_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.ListByUser("u1", "bad")
	assert.ErrorContains(t, err, "invalid status")
}

func TestStore_ListByUser_Empty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	list, err := store.ListByUser("u1", "")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestStore_ListByUser_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.ListByUser("", "")
	assert.ErrorContains(t, err, "user_id is required")
}

// --- Update ---

func TestStore_Update_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	ui := &UserIntegration{UserID: "u1", IntegrationID: "google"}
	require.NoError(t, store.Create(ui))

	ui.Status = UserIntegrationActive
	ui.Config = map[string]string{"new": "config"}
	require.NoError(t, store.Update(ui))

	got, _ := store.Get("u1", "google")
	assert.Equal(t, UserIntegrationActive, got.Status)
	assert.Equal(t, "config", got.Config["new"])
}

func TestStore_Update_Nil(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Update(nil), "nil")
}

func TestStore_Update_EmptyID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Update(&UserIntegration{}), "id is required")
}

func TestStore_Update_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.Update(&UserIntegration{ID: "nonexistent", Status: UserIntegrationActive})
	assert.ErrorContains(t, err, "not found")
}

func TestStore_Update_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.ErrorContains(t, store.Update(&UserIntegration{ID: "x", Status: "bad"}), "invalid status")
}

// --- UpdateStatus ---

func TestStore_UpdateStatus_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))

	err := store.UpdateStatus("u1", "google", UserIntegrationActive, "")
	require.NoError(t, err)

	got, _ := store.Get("u1", "google")
	assert.Equal(t, UserIntegrationActive, got.Status)
}

func TestStore_UpdateStatus_WithError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))

	err := store.UpdateStatus("u1", "google", UserIntegrationFailed, "token expired")
	require.NoError(t, err)

	got, _ := store.Get("u1", "google")
	assert.Equal(t, UserIntegrationFailed, got.Status)
	assert.Equal(t, "token expired", got.ErrorMessage)
}

func TestStore_UpdateStatus_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.UpdateStatus("u1", "nonexistent", UserIntegrationActive, "")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_UpdateStatus_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.UpdateStatus("u1", "google", "bad", "")
	assert.ErrorContains(t, err, "invalid status")
}

func TestStore_UpdateStatus_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.UpdateStatus("", "google", UserIntegrationActive, "")
	assert.ErrorContains(t, err, "user_id is required")
}

func TestStore_UpdateStatus_EmptyIntegrationID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.UpdateStatus("u1", "", UserIntegrationActive, "")
	assert.ErrorContains(t, err, "integration_id is required")
}

// --- Delete ---

func TestStore_Delete_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))

	err := store.Delete("u1", "google")
	require.NoError(t, err)

	_, err = store.Get("u1", "google")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_Delete_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.Delete("u1", "nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_Delete_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.Delete("", "google")
	assert.ErrorContains(t, err, "user_id is required")
}

func TestStore_Delete_EmptyIntegrationID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.Delete("u1", "")
	assert.ErrorContains(t, err, "integration_id is required")
}

// --- RecordUsage ---

func TestStore_RecordUsage_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))

	err := store.RecordUsage("u1", "google")
	require.NoError(t, err)

	got, _ := store.Get("u1", "google")
	assert.NotNil(t, got.LastUsedAt)
	assert.WithinDuration(t, time.Now(), *got.LastUsedAt, 5*time.Second)
}

func TestStore_RecordUsage_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.RecordUsage("u1", "nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestStore_RecordUsage_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	err := store.RecordUsage("", "google")
	assert.ErrorContains(t, err, "user_id is required")
}

// --- CountByUser ---

func TestStore_CountByUser_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "shopify"}))

	count, err := store.CountByUser("u1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestStore_CountByUser_Empty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	count, err := store.CountByUser("u1")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestStore_CountByUser_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	_, err := store.CountByUser("")
	assert.ErrorContains(t, err, "user_id is required")
}

// --- Multi-user isolation ---

func TestStore_MultiUserIsolation(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)

	require.NoError(t, store.Create(&UserIntegration{UserID: "u1", IntegrationID: "google"}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u2", IntegrationID: "google"}))
	require.NoError(t, store.Create(&UserIntegration{UserID: "u2", IntegrationID: "shopify"}))

	list1, _ := store.ListByUser("u1", "")
	assert.Len(t, list1, 1)

	list2, _ := store.ListByUser("u2", "")
	assert.Len(t, list2, 2)

	count1, _ := store.CountByUser("u1")
	assert.Equal(t, 1, count1)

	count2, _ := store.CountByUser("u2")
	assert.Equal(t, 2, count2)

	// Delete from u1 doesn't affect u2
	require.NoError(t, store.Delete("u1", "google"))
	got, err := store.Get("u2", "google")
	require.NoError(t, err)
	assert.Equal(t, "u2", got.UserID)
}

// --- Close ---

func TestStore_Close(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)
	assert.NoError(t, store.Close())
}

// --- ValidUserIntegrationStatus ---

func TestValidUserIntegrationStatus(t *testing.T) {
	assert.True(t, ValidUserIntegrationStatus(UserIntegrationPending))
	assert.True(t, ValidUserIntegrationStatus(UserIntegrationActive))
	assert.True(t, ValidUserIntegrationStatus(UserIntegrationFailed))
	assert.True(t, ValidUserIntegrationStatus(UserIntegrationRevoked))
	assert.True(t, ValidUserIntegrationStatus(UserIntegrationDisabled))
	assert.False(t, ValidUserIntegrationStatus("bad"))
	assert.False(t, ValidUserIntegrationStatus(""))
}

// --- Full lifecycle ---

func TestStore_FullLifecycle(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, _ := NewSQLiteUserIntegrationStore(db)

	// Create pending
	ui := &UserIntegration{
		UserID:        "u1",
		IntegrationID: "google",
		Config:        map[string]string{"domain": "gmail.com"},
		Scopes:        []string{"email", "calendar"},
	}
	require.NoError(t, store.Create(ui))
	assert.Equal(t, UserIntegrationPending, ui.Status)

	// Activate
	require.NoError(t, store.UpdateStatus("u1", "google", UserIntegrationActive, ""))
	got, _ := store.Get("u1", "google")
	assert.Equal(t, UserIntegrationActive, got.Status)

	// Record usage
	require.NoError(t, store.RecordUsage("u1", "google"))
	got, _ = store.Get("u1", "google")
	assert.NotNil(t, got.LastUsedAt)

	// Update config
	got.Config["new_key"] = "new_value"
	require.NoError(t, store.Update(got))
	got2, _ := store.Get("u1", "google")
	assert.Equal(t, "new_value", got2.Config["new_key"])

	// Fail
	require.NoError(t, store.UpdateStatus("u1", "google", UserIntegrationFailed, "token expired"))
	got3, _ := store.Get("u1", "google")
	assert.Equal(t, UserIntegrationFailed, got3.Status)
	assert.Equal(t, "token expired", got3.ErrorMessage)

	// Delete
	require.NoError(t, store.Delete("u1", "google"))
	_, err := store.Get("u1", "google")
	assert.ErrorContains(t, err, "not found")
}
