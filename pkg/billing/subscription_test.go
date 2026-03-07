package billing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func testSub() *Subscription {
	now := time.Now().UTC().Truncate(time.Second)
	return &Subscription{
		ID:                 "sub_001",
		UserID:             "user_001",
		PlanID:             PlanFree,
		Status:             SubStatusActive,
		BillingInterval:    IntervalNone,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
}

// ---------- Store creation ----------

func TestNewSQLiteSubscriptionStoreNilDB(t *testing.T) {
	_, err := NewSQLiteSubscriptionStore(nil)
	assert.Error(t, err)
}

func TestNewSQLiteSubscriptionStore(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteSubscriptionStore(db)
	require.NoError(t, err)
	require.NotNil(t, store)
}

// ---------- Create ----------

func TestCreateSubscription(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	err := store.Create(sub)
	require.NoError(t, err)
	assert.False(t, sub.CreatedAt.IsZero())
	assert.False(t, sub.UpdatedAt.IsZero())
}

func TestCreateDuplicateID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	require.NoError(t, store.Create(sub))

	err := store.Create(sub)
	assert.ErrorIs(t, err, ErrDuplicateID)
}

func TestCreateNilSubscription(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	assert.Error(t, store.Create(nil))
}

func TestCreateEmptyID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	sub.ID = ""
	assert.Error(t, store.Create(sub))
}

func TestCreateInvalidPlan(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	sub.PlanID = "invalid"
	assert.ErrorIs(t, store.Create(sub), ErrInvalidPlan)
}

func TestCreateInvalidStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	sub.Status = "bogus"
	assert.ErrorIs(t, store.Create(sub), ErrInvalidStatus)
}

func TestCreateDefaultsStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	sub.Status = "" // should default to active
	require.NoError(t, store.Create(sub))

	got, err := store.GetByID(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, SubStatusActive, got.Status)
}

// ---------- GetByID ----------

func TestGetByID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	require.NoError(t, store.Create(sub))

	got, err := store.GetByID(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, sub.ID, got.ID)
	assert.Equal(t, sub.UserID, got.UserID)
	assert.Equal(t, PlanFree, got.PlanID)
	assert.Equal(t, SubStatusActive, got.Status)
	assert.Equal(t, IntervalNone, got.BillingInterval)
	assert.False(t, got.CancelAtPeriodEnd)
}

func TestGetByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	_, err := store.GetByID("nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------- GetByUserID ----------

func TestGetByUserID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	require.NoError(t, store.Create(sub))

	got, err := store.GetByUserID("user_001")
	require.NoError(t, err)
	assert.Equal(t, sub.ID, got.ID)
}

func TestGetByUserIDReturnsLatest(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	now := time.Now().UTC().Truncate(time.Second)

	sub1 := &Subscription{
		ID: "sub_old", UserID: "user_001", PlanID: PlanFree,
		Status: SubStatusCanceled, BillingInterval: IntervalNone,
		CurrentPeriodStart: now.Add(-60 * 24 * time.Hour),
		CurrentPeriodEnd:   now.Add(-30 * 24 * time.Hour),
		CreatedAt:          now.Add(-60 * 24 * time.Hour),
	}
	sub2 := &Subscription{
		ID: "sub_new", UserID: "user_001", PlanID: PlanStarter,
		Status: SubStatusActive, BillingInterval: IntervalMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	require.NoError(t, store.Create(sub1))
	require.NoError(t, store.Create(sub2))

	got, err := store.GetByUserID("user_001")
	require.NoError(t, err)
	assert.Equal(t, "sub_new", got.ID)
}

func TestGetByUserIDNotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	_, err := store.GetByUserID("nobody")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------- Update ----------

func TestUpdate(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	require.NoError(t, store.Create(sub))

	sub.PlanID = PlanPro
	sub.Status = SubStatusActive
	sub.BillingInterval = IntervalMonthly
	sub.CancelAtPeriodEnd = true
	sub.StripeCustomerID = "cus_123"
	sub.StripeSubID = "sub_stripe_123"

	err := store.Update(sub)
	require.NoError(t, err)

	got, _ := store.GetByID(sub.ID)
	assert.Equal(t, PlanPro, got.PlanID)
	assert.True(t, got.CancelAtPeriodEnd)
	assert.Equal(t, "cus_123", got.StripeCustomerID)
	assert.Equal(t, "sub_stripe_123", got.StripeSubID)
	assert.Equal(t, IntervalMonthly, got.BillingInterval)
}

func TestUpdateNotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	assert.ErrorIs(t, store.Update(sub), ErrNotFound)
}

func TestUpdateNil(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	assert.Error(t, store.Update(nil))
}

func TestUpdateInvalidStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	require.NoError(t, store.Create(sub))
	sub.Status = "invalid"
	assert.ErrorIs(t, store.Update(sub), ErrInvalidStatus)
}

func TestUpdateInvalidPlan(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	sub := testSub()
	require.NoError(t, store.Create(sub))
	sub.PlanID = "invalid"
	assert.ErrorIs(t, store.Update(sub), ErrInvalidPlan)
}

// ---------- ListByStatus ----------

func TestListByStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	for i, status := range []SubscriptionStatus{SubStatusActive, SubStatusActive, SubStatusCanceled} {
		sub := &Subscription{
			ID: fmt.Sprintf("sub_%d", i), UserID: fmt.Sprintf("user_%d", i),
			PlanID: PlanFree, Status: status, BillingInterval: IntervalNone,
			CurrentPeriodStart: now, CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
		}
		require.NoError(t, store.Create(sub))
	}

	active, err := store.ListByStatus(SubStatusActive)
	require.NoError(t, err)
	assert.Len(t, active, 2)

	canceled, err := store.ListByStatus(SubStatusCanceled)
	require.NoError(t, err)
	assert.Len(t, canceled, 1)
}

func TestListByStatusEmpty(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	list, err := store.ListByStatus(SubStatusActive)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestListByStatusInvalid(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	_, err := store.ListByStatus("invalid")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

// ---------- IsActive ----------

func TestSubscriptionIsActive(t *testing.T) {
	tests := []struct {
		status SubscriptionStatus
		want   bool
	}{
		{SubStatusActive, true},
		{SubStatusTrialing, true},
		{SubStatusPastDue, true},
		{SubStatusCanceled, false},
		{SubStatusExpired, false},
		{SubStatusPaused, false},
	}
	for _, tt := range tests {
		sub := &Subscription{Status: tt.status}
		assert.Equal(t, tt.want, sub.IsActive(), "IsActive() for status %q", tt.status)
	}
}

// ---------- ValidSubscriptionStatus ----------

func TestValidSubscriptionStatus(t *testing.T) {
	tests := []struct {
		s    SubscriptionStatus
		want bool
	}{
		{SubStatusActive, true},
		{SubStatusTrialing, true},
		{SubStatusPastDue, true},
		{SubStatusCanceled, true},
		{SubStatusExpired, true},
		{SubStatusPaused, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ValidSubscriptionStatus(tt.s), "ValidSubscriptionStatus(%q)", tt.s)
	}
}

// ---------- Persistence ----------

func TestPersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db1, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	db1.Exec("PRAGMA journal_mode=WAL")
	store1, _ := NewSQLiteSubscriptionStore(db1)

	sub := testSub()
	require.NoError(t, store1.Create(sub))
	db1.Close()

	// Reopen.
	db2, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db2.Close()
	store2, _ := NewSQLiteSubscriptionStore(db2)

	got, err := store2.GetByID(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, sub.UserID, got.UserID)
	assert.Equal(t, PlanFree, got.PlanID)
}

// ---------- Stripe fields ----------

func TestStripeFields(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)

	sub := testSub()
	sub.PlanID = PlanStarter
	sub.BillingInterval = IntervalMonthly
	sub.StripeCustomerID = "cus_abc"
	sub.StripeSubID = "sub_xyz"
	require.NoError(t, store.Create(sub))

	got, _ := store.GetByID(sub.ID)
	assert.Equal(t, "cus_abc", got.StripeCustomerID)
	assert.Equal(t, "sub_xyz", got.StripeSubID)
}

// ---------- Close ----------

func TestStoreClose(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewSQLiteSubscriptionStore(db)
	assert.NoError(t, store.Close())
}

// ---------- fmt import for Sprintf in ListByStatus test ----------

func init() {
	// Ensure temp dirs are cleaned up properly.
	_ = os.TempDir()
}
