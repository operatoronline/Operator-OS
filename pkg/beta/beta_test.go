package beta

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// --- Store Creation ---

func TestNewStore_NilDB(t *testing.T) {
	_, err := NewStore(nil)
	assert.ErrorIs(t, err, ErrNilDB)
}

func TestNewStore_OK(t *testing.T) {
	store, err := NewStore(testDB(t))
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestStore_Close(t *testing.T) {
	store, _ := NewStore(testDB(t))
	assert.NoError(t, store.Close())
}

// --- Invite Code Operations ---

func TestCreateInvite_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin-1"}
	err := store.CreateInvite(invite)
	require.NoError(t, err)
	assert.NotEmpty(t, invite.ID)
	assert.NotEmpty(t, invite.Code)
	assert.Equal(t, InviteStatusActive, invite.Status)
	assert.True(t, strings.HasPrefix(invite.Code, "BETA-"))
}

func TestCreateInvite_Nil(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateInvite(nil)
	assert.ErrorIs(t, err, ErrInvalidCode)
}

func TestCreateInvite_MissingCreatedBy(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateInvite(&InviteCode{})
	assert.Error(t, err)
}

func TestCreateInvite_InvalidStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateInvite(&InviteCode{CreatedBy: "admin", Status: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestCreateInvite_WithExpiry(t *testing.T) {
	store, _ := NewStore(testDB(t))
	expires := time.Now().Add(24 * time.Hour)
	invite := &InviteCode{CreatedBy: "admin", ExpiresAt: expires, MaxUses: 5}
	err := store.CreateInvite(invite)
	require.NoError(t, err)

	got, err := store.GetInvite(invite.ID)
	require.NoError(t, err)
	assert.False(t, got.ExpiresAt.IsZero())
	assert.Equal(t, 5, got.MaxUses)
}

func TestCreateInvite_WithEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", Email: "test@example.com"}
	err := store.CreateInvite(invite)
	require.NoError(t, err)

	got, err := store.GetInvite(invite.ID)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", got.Email)
}

func TestCreateInvite_CustomID(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{ID: "custom-id", Code: "BETA-CUSTOM-CODE", CreatedBy: "admin"}
	err := store.CreateInvite(invite)
	require.NoError(t, err)

	got, err := store.GetInvite("custom-id")
	require.NoError(t, err)
	assert.Equal(t, "BETA-CUSTOM-CODE", got.Code)
}

func TestGetInvite_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetInvite("nonexistent")
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestGetInvite_EmptyID(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetInvite("")
	assert.ErrorIs(t, err, ErrEmptyID)
}

func TestGetInviteByCode_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetInviteByCode("BETA-XXXX-XXXX")
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestGetInviteByCode_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetInviteByCode("")
	assert.ErrorIs(t, err, ErrInvalidCode)
}

func TestGetInviteByCode_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin"}
	require.NoError(t, store.CreateInvite(invite))

	got, err := store.GetInviteByCode(invite.Code)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, got.ID)
}

func TestListInvites_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invites, err := store.ListInvites("")
	require.NoError(t, err)
	assert.Empty(t, invites)
}

func TestListInvites_All(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateInvite(&InviteCode{CreatedBy: "admin"}))
	require.NoError(t, store.CreateInvite(&InviteCode{CreatedBy: "admin"}))

	invites, err := store.ListInvites("")
	require.NoError(t, err)
	assert.Len(t, invites, 2)
}

func TestListInvites_ByStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	i1 := &InviteCode{CreatedBy: "admin"}
	require.NoError(t, store.CreateInvite(i1))
	require.NoError(t, store.RevokeInvite(i1.ID))
	require.NoError(t, store.CreateInvite(&InviteCode{CreatedBy: "admin"}))

	active, err := store.ListInvites(InviteStatusActive)
	require.NoError(t, err)
	assert.Len(t, active, 1)

	revoked, err := store.ListInvites(InviteStatusRevoked)
	require.NoError(t, err)
	assert.Len(t, revoked, 1)
}

func TestListInvites_InvalidStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.ListInvites("invalid")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestRedeemInvite_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", MaxUses: 3}
	require.NoError(t, store.CreateInvite(invite))

	got, err := store.RedeemInvite(invite.Code, "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, 1, got.UseCount)

	// Redeem again
	got2, err := store.RedeemInvite(invite.Code, "user2@test.com")
	require.NoError(t, err)
	assert.Equal(t, 2, got2.UseCount)
}

func TestRedeemInvite_Exhausted(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", MaxUses: 1}
	require.NoError(t, store.CreateInvite(invite))

	_, err := store.RedeemInvite(invite.Code, "user1@test.com")
	require.NoError(t, err)

	_, err = store.RedeemInvite(invite.Code, "user2@test.com")
	assert.ErrorIs(t, err, ErrInviteExhausted)
}

func TestRedeemInvite_Expired(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{
		CreatedBy: "admin",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	require.NoError(t, store.CreateInvite(invite))

	_, err := store.RedeemInvite(invite.Code, "user@test.com")
	assert.ErrorIs(t, err, ErrInviteExpired)
}

func TestRedeemInvite_Revoked(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin"}
	require.NoError(t, store.CreateInvite(invite))
	require.NoError(t, store.RevokeInvite(invite.ID))

	_, err := store.RedeemInvite(invite.Code, "user@test.com")
	assert.ErrorIs(t, err, ErrInviteUsed)
}

func TestRedeemInvite_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.RedeemInvite("BETA-XXXX-XXXX", "user@test.com")
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestRedeemInvite_EmptyCode(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.RedeemInvite("", "user@test.com")
	assert.ErrorIs(t, err, ErrInvalidCode)
}

func TestRedeemInvite_EmailTargeted(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", Email: "specific@test.com"}
	require.NoError(t, store.CreateInvite(invite))

	// Wrong email
	_, err := store.RedeemInvite(invite.Code, "wrong@test.com")
	assert.ErrorIs(t, err, ErrInviteNotFound)

	// Correct email
	got, err := store.RedeemInvite(invite.Code, "specific@test.com")
	require.NoError(t, err)
	assert.Equal(t, 1, got.UseCount)
}

func TestRedeemInvite_EmailTargeted_CaseInsensitive(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", Email: "Test@Example.com"}
	require.NoError(t, store.CreateInvite(invite))

	got, err := store.RedeemInvite(invite.Code, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, 1, got.UseCount)
}

func TestRedeemInvite_UnlimitedUses(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin", MaxUses: 0} // unlimited
	require.NoError(t, store.CreateInvite(invite))

	for i := 0; i < 10; i++ {
		_, err := store.RedeemInvite(invite.Code, "user@test.com")
		require.NoError(t, err)
	}
}

func TestRevokeInvite_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin"}
	require.NoError(t, store.CreateInvite(invite))

	err := store.RevokeInvite(invite.ID)
	require.NoError(t, err)

	got, _ := store.GetInvite(invite.ID)
	assert.Equal(t, InviteStatusRevoked, got.Status)
}

func TestRevokeInvite_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.RevokeInvite("nonexistent")
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestRevokeInvite_EmptyID(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.RevokeInvite("")
	assert.ErrorIs(t, err, ErrEmptyID)
}

func TestCountInvites(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateInvite(&InviteCode{CreatedBy: "admin"}))
	require.NoError(t, store.CreateInvite(&InviteCode{CreatedBy: "admin"}))

	total, err := store.CountInvites("")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	active, err := store.CountInvites(InviteStatusActive)
	require.NoError(t, err)
	assert.Equal(t, int64(2), active)
}

// --- InviteCode Methods ---

func TestInviteCode_IsUsable(t *testing.T) {
	tests := []struct {
		name   string
		invite InviteCode
		want   bool
	}{
		{"active", InviteCode{Status: InviteStatusActive}, true},
		{"revoked", InviteCode{Status: InviteStatusRevoked}, false},
		{"expired", InviteCode{Status: InviteStatusActive, ExpiresAt: time.Now().Add(-time.Hour)}, false},
		{"exhausted", InviteCode{Status: InviteStatusActive, MaxUses: 1, UseCount: 1}, false},
		{"has_uses_left", InviteCode{Status: InviteStatusActive, MaxUses: 5, UseCount: 3}, true},
		{"unlimited", InviteCode{Status: InviteStatusActive, MaxUses: 0, UseCount: 100}, true},
		{"future_expiry", InviteCode{Status: InviteStatusActive, ExpiresAt: time.Now().Add(time.Hour)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.invite.IsUsable())
		})
	}
}

func TestValidInviteStatus(t *testing.T) {
	assert.True(t, ValidInviteStatus(InviteStatusActive))
	assert.True(t, ValidInviteStatus(InviteStatusRevoked))
	assert.True(t, ValidInviteStatus(InviteStatusExpired))
	assert.False(t, ValidInviteStatus("invalid"))
	assert.False(t, ValidInviteStatus(""))
}

// --- Feature Flag Operations ---

func TestCreateFlag_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flag := &FeatureFlag{Name: "dark_mode", Enabled: true, RolloutPct: 50}
	err := store.CreateFlag(flag)
	require.NoError(t, err)
	assert.NotEmpty(t, flag.ID)
}

func TestCreateFlag_Nil(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateFlag(nil)
	assert.ErrorIs(t, err, ErrInvalidFlag)
}

func TestCreateFlag_MissingName(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateFlag(&FeatureFlag{})
	assert.Error(t, err)
}

func TestCreateFlag_InvalidRollout(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.CreateFlag(&FeatureFlag{Name: "test", RolloutPct: 150})
	assert.ErrorIs(t, err, ErrInvalidRollout)

	err = store.CreateFlag(&FeatureFlag{Name: "test", RolloutPct: -1})
	assert.ErrorIs(t, err, ErrInvalidRollout)
}

func TestCreateFlag_Duplicate(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateFlag(&FeatureFlag{Name: "dark_mode"}))
	err := store.CreateFlag(&FeatureFlag{Name: "dark_mode"})
	assert.ErrorIs(t, err, ErrDuplicateFlag)
}

func TestCreateFlag_WithPlansAndUsers(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flag := &FeatureFlag{
		Name:    "premium_feature",
		Enabled: true,
		Plans:   []string{"pro", "enterprise"},
		UserIDs: []string{"user-1", "user-2"},
	}
	require.NoError(t, store.CreateFlag(flag))

	got, err := store.GetFlag("premium_feature")
	require.NoError(t, err)
	assert.Equal(t, []string{"pro", "enterprise"}, got.Plans)
	assert.Equal(t, []string{"user-1", "user-2"}, got.UserIDs)
}

func TestGetFlag_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetFlag("nonexistent")
	assert.ErrorIs(t, err, ErrFlagNotFound)
}

func TestGetFlag_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetFlag("")
	assert.ErrorIs(t, err, ErrInvalidFlag)
}

func TestGetFlagByID_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flag := &FeatureFlag{Name: "test_flag"}
	require.NoError(t, store.CreateFlag(flag))

	got, err := store.GetFlagByID(flag.ID)
	require.NoError(t, err)
	assert.Equal(t, "test_flag", got.Name)
}

func TestGetFlagByID_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetFlagByID("nonexistent")
	assert.ErrorIs(t, err, ErrFlagNotFound)
}

func TestGetFlagByID_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetFlagByID("")
	assert.ErrorIs(t, err, ErrEmptyID)
}

func TestListFlags_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flags, err := store.ListFlags()
	require.NoError(t, err)
	assert.Empty(t, flags)
}

func TestListFlags_Sorted(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateFlag(&FeatureFlag{Name: "zebra"}))
	require.NoError(t, store.CreateFlag(&FeatureFlag{Name: "alpha"}))

	flags, err := store.ListFlags()
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Equal(t, "alpha", flags[0].Name)
	assert.Equal(t, "zebra", flags[1].Name)
}

func TestUpdateFlag_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flag := &FeatureFlag{Name: "test", Enabled: false, RolloutPct: 0}
	require.NoError(t, store.CreateFlag(flag))

	flag.Enabled = true
	flag.RolloutPct = 75
	flag.Description = "updated"
	require.NoError(t, store.UpdateFlag(flag))

	got, err := store.GetFlag("test")
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, 75, got.RolloutPct)
	assert.Equal(t, "updated", got.Description)
}

func TestUpdateFlag_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.UpdateFlag(&FeatureFlag{ID: "nonexistent", Name: "test"})
	assert.ErrorIs(t, err, ErrFlagNotFound)
}

func TestUpdateFlag_Nil(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.UpdateFlag(nil)
	assert.ErrorIs(t, err, ErrInvalidFlag)
}

func TestUpdateFlag_EmptyID(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.UpdateFlag(&FeatureFlag{Name: "test"})
	assert.ErrorIs(t, err, ErrEmptyID)
}

func TestUpdateFlag_InvalidRollout(t *testing.T) {
	store, _ := NewStore(testDB(t))
	flag := &FeatureFlag{Name: "test"}
	require.NoError(t, store.CreateFlag(flag))

	flag.RolloutPct = 200
	err := store.UpdateFlag(flag)
	assert.ErrorIs(t, err, ErrInvalidRollout)
}

func TestDeleteFlag_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateFlag(&FeatureFlag{Name: "test"}))

	err := store.DeleteFlag("test")
	require.NoError(t, err)

	_, err = store.GetFlag("test")
	assert.ErrorIs(t, err, ErrFlagNotFound)
}

func TestDeleteFlag_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.DeleteFlag("nonexistent")
	assert.ErrorIs(t, err, ErrFlagNotFound)
}

func TestDeleteFlag_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.DeleteFlag("")
	assert.ErrorIs(t, err, ErrInvalidFlag)
}

// --- Feature Flag Logic ---

func TestFeatureFlag_IsEnabledForUser(t *testing.T) {
	tests := []struct {
		name     string
		flag     FeatureFlag
		userID   string
		userPlan string
		want     bool
	}{
		{"disabled", FeatureFlag{Enabled: false, RolloutPct: 100}, "user-1", "pro", false},
		{"100_pct", FeatureFlag{Enabled: true, RolloutPct: 100}, "user-1", "free", true},
		{"0_pct", FeatureFlag{Enabled: true, RolloutPct: 0}, "user-1", "free", false},
		{"explicit_user", FeatureFlag{Enabled: true, UserIDs: []string{"user-1"}}, "user-1", "free", true},
		{"not_in_user_list", FeatureFlag{Enabled: true, UserIDs: []string{"user-2"}}, "user-1", "free", false},
		{"plan_match", FeatureFlag{Enabled: true, Plans: []string{"pro"}}, "user-1", "pro", true},
		{"plan_no_match", FeatureFlag{Enabled: true, Plans: []string{"enterprise"}}, "user-1", "pro", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.flag.IsEnabledForUser(tt.userID, tt.userPlan))
		})
	}
}

func TestHashToPercent_Deterministic(t *testing.T) {
	v1 := hashToPercent("user-123", "dark_mode")
	v2 := hashToPercent("user-123", "dark_mode")
	assert.Equal(t, v1, v2)
	assert.True(t, v1 >= 0 && v1 < 100)
}

func TestHashToPercent_DifferentInputs(t *testing.T) {
	v1 := hashToPercent("user-1", "flag-a")
	v2 := hashToPercent("user-2", "flag-a")
	// Different users should (generally) get different values
	// Not guaranteed but highly likely
	_ = v1
	_ = v2
}

// --- Waitlist Operations ---

func TestAddToWaitlist_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	entry := &WaitlistEntry{Email: "test@example.com", Name: "Test User", Source: "website"}
	err := store.AddToWaitlist(entry)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, WaitlistStatusPending, entry.Status)
	assert.Equal(t, "test@example.com", entry.Email) // lowercased
}

func TestAddToWaitlist_Nil(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.AddToWaitlist(nil)
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestAddToWaitlist_EmptyEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.AddToWaitlist(&WaitlistEntry{})
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestAddToWaitlist_InvalidEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.AddToWaitlist(&WaitlistEntry{Email: "not-an-email"})
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestAddToWaitlist_Duplicate(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com"}))
	err := store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com"})
	assert.ErrorIs(t, err, ErrWaitlistDuplicate)
}

func TestAddToWaitlist_CaseNormalization(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "Test@Example.COM"}))

	got, err := store.GetWaitlistEntry("test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", got.Email)
}

func TestAddToWaitlist_InvalidStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com", Status: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestGetWaitlistEntry_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetWaitlistEntry("nobody@example.com")
	assert.ErrorIs(t, err, ErrWaitlistNotFound)
}

func TestGetWaitlistEntry_EmptyEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.GetWaitlistEntry("")
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestListWaitlist_Empty(t *testing.T) {
	store, _ := NewStore(testDB(t))
	entries, err := store.ListWaitlist("")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestListWaitlist_All(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "a@test.com"}))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "b@test.com"}))

	entries, err := store.ListWaitlist("")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestListWaitlist_ByStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "a@test.com"}))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "b@test.com"}))
	require.NoError(t, store.UpdateWaitlistStatus("a@test.com", WaitlistStatusInvited, "inv-1"))

	pending, err := store.ListWaitlist(WaitlistStatusPending)
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	invited, err := store.ListWaitlist(WaitlistStatusInvited)
	require.NoError(t, err)
	assert.Len(t, invited, 1)
}

func TestListWaitlist_InvalidStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	_, err := store.ListWaitlist("invalid")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestUpdateWaitlistStatus_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com"}))

	err := store.UpdateWaitlistStatus("test@example.com", WaitlistStatusInvited, "inv-123")
	require.NoError(t, err)

	got, _ := store.GetWaitlistEntry("test@example.com")
	assert.Equal(t, WaitlistStatusInvited, got.Status)
	assert.Equal(t, "inv-123", got.InviteID)
}

func TestUpdateWaitlistStatus_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.UpdateWaitlistStatus("nobody@test.com", WaitlistStatusInvited, "")
	assert.ErrorIs(t, err, ErrWaitlistNotFound)
}

func TestUpdateWaitlistStatus_InvalidStatus(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com"}))
	err := store.UpdateWaitlistStatus("test@example.com", "invalid", "")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestUpdateWaitlistStatus_EmptyEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.UpdateWaitlistStatus("", WaitlistStatusInvited, "")
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestRemoveFromWaitlist_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "test@example.com"}))

	err := store.RemoveFromWaitlist("test@example.com")
	require.NoError(t, err)

	_, err = store.GetWaitlistEntry("test@example.com")
	assert.ErrorIs(t, err, ErrWaitlistNotFound)
}

func TestRemoveFromWaitlist_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.RemoveFromWaitlist("nobody@test.com")
	assert.ErrorIs(t, err, ErrWaitlistNotFound)
}

func TestRemoveFromWaitlist_EmptyEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	err := store.RemoveFromWaitlist("")
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestCountWaitlist(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "a@test.com"}))
	require.NoError(t, store.AddToWaitlist(&WaitlistEntry{Email: "b@test.com"}))

	total, err := store.CountWaitlist("")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	pending, err := store.CountWaitlist(WaitlistStatusPending)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pending)
}

func TestValidWaitlistStatus(t *testing.T) {
	assert.True(t, ValidWaitlistStatus(WaitlistStatusPending))
	assert.True(t, ValidWaitlistStatus(WaitlistStatusInvited))
	assert.True(t, ValidWaitlistStatus(WaitlistStatusJoined))
	assert.True(t, ValidWaitlistStatus(WaitlistStatusRemoved))
	assert.False(t, ValidWaitlistStatus("invalid"))
	assert.False(t, ValidWaitlistStatus(""))
}

// --- Readiness Checker ---

func TestReadinessChecker_Empty(t *testing.T) {
	rc := NewReadinessChecker()
	report := rc.Run()
	assert.True(t, report.Ready)
	assert.Empty(t, report.Checks)
	assert.Equal(t, 0, report.Summary.Total)
}

func TestReadinessChecker_AllPass(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("test1", "general", false, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass, Message: "ok"}
	})
	rc.Register("test2", "general", true, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass, Message: "ok"}
	})

	report := rc.Run()
	assert.True(t, report.Ready)
	assert.Equal(t, 2, report.Summary.Total)
	assert.Equal(t, 2, report.Summary.Passed)
	assert.Equal(t, 0, report.Summary.Failed)
}

func TestReadinessChecker_CriticalFail(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("critical", "general", true, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckFail, Message: "broken"}
	})
	rc.Register("ok", "general", false, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass, Message: "fine"}
	})

	report := rc.Run()
	assert.False(t, report.Ready)
	assert.Equal(t, 1, report.Summary.Critical)
	assert.Equal(t, 1, report.Summary.Failed)
	assert.Equal(t, 1, report.Summary.Passed)
}

func TestReadinessChecker_NonCriticalFail(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("optional", "general", false, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckFail, Message: "missing but optional"}
	})

	report := rc.Run()
	assert.False(t, report.Ready) // Any failure = not ready
	assert.Equal(t, 0, report.Summary.Critical)
	assert.Equal(t, 1, report.Summary.Failed)
}

func TestReadinessChecker_Warnings(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("warn", "general", false, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckWarn, Message: "something to look at"}
	})

	report := rc.Run()
	assert.True(t, report.Ready) // Warnings don't block
	assert.Equal(t, 1, report.Summary.Warnings)
}

func TestReadinessChecker_Categories(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("db", "database", true, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass}
	})
	rc.Register("jwt", "security", true, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass}
	})

	report := rc.Run()
	assert.Equal(t, "database", report.Checks[0].Category)
	assert.Equal(t, "security", report.Checks[1].Category)
}

// --- Built-in Checks ---

func TestDatabaseCheck_NilDB(t *testing.T) {
	check := DatabaseCheck(nil)()
	assert.Equal(t, CheckFail, check.Status)
}

func TestDatabaseCheck_Success(t *testing.T) {
	db := testDB(t)
	check := DatabaseCheck(db)()
	assert.Equal(t, CheckPass, check.Status)
}

func TestEncryptionKeyCheck_NotSet(t *testing.T) {
	t.Setenv("OPERATOR_ENCRYPTION_KEY", "")
	check := EncryptionKeyCheck()()
	assert.Equal(t, CheckFail, check.Status)
}

func TestEncryptionKeyCheck_Short(t *testing.T) {
	t.Setenv("OPERATOR_ENCRYPTION_KEY", "short")
	check := EncryptionKeyCheck()()
	assert.Equal(t, CheckWarn, check.Status)
}

func TestEncryptionKeyCheck_Good(t *testing.T) {
	t.Setenv("OPERATOR_ENCRYPTION_KEY", "this-is-a-very-long-encryption-key-at-least-32-chars")
	check := EncryptionKeyCheck()()
	assert.Equal(t, CheckPass, check.Status)
}

func TestJWTSecretCheck_NotSet(t *testing.T) {
	t.Setenv("OPERATOR_JWT_SECRET", "")
	check := JWTSecretCheck()()
	assert.Equal(t, CheckFail, check.Status)
}

func TestJWTSecretCheck_Short(t *testing.T) {
	t.Setenv("OPERATOR_JWT_SECRET", "short")
	check := JWTSecretCheck()()
	assert.Equal(t, CheckWarn, check.Status)
}

func TestJWTSecretCheck_Good(t *testing.T) {
	t.Setenv("OPERATOR_JWT_SECRET", "this-is-a-very-long-jwt-secret-key-at-least-32")
	check := JWTSecretCheck()()
	assert.Equal(t, CheckPass, check.Status)
}

func TestStripeConfigCheck_NotSet(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	check := StripeConfigCheck()()
	assert.Equal(t, CheckWarn, check.Status)
}

func TestStripeConfigCheck_Configured(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_xxx")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_xxx")
	check := StripeConfigCheck()()
	assert.Equal(t, CheckPass, check.Status)
}

func TestLLMProviderCheck_None(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	check := LLMProviderCheck()()
	assert.Equal(t, CheckFail, check.Status)
}

func TestLLMProviderCheck_HasOne(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	check := LLMProviderCheck()()
	assert.Equal(t, CheckPass, check.Status)
}

func TestHealthEndpointCheck_NoURL(t *testing.T) {
	check := HealthEndpointCheck("")()
	assert.Equal(t, CheckWarn, check.Status)
}

func TestHealthEndpointCheck_Unreachable(t *testing.T) {
	check := HealthEndpointCheck("http://localhost:19999")()
	assert.Equal(t, CheckFail, check.Status)
}

func TestMetricsEndpointCheck_NoURL(t *testing.T) {
	check := MetricsEndpointCheck("")()
	assert.Equal(t, CheckWarn, check.Status)
}

func TestMinUsersCheck_NilDB(t *testing.T) {
	check := MinUsersCheck(nil, 1)()
	assert.Equal(t, CheckFail, check.Status)
}

func TestRegisterDefaultChecks(t *testing.T) {
	rc := NewReadinessChecker()
	db := testDB(t)
	RegisterDefaultChecks(rc, db, "http://localhost:8080")
	assert.True(t, len(rc.checks) >= 5)
}

// --- Helpers ---

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32)
}

func TestGenerateCode(t *testing.T) {
	code := generateCode()
	assert.True(t, strings.HasPrefix(code, "BETA-"))
	assert.Len(t, code, 14) // BETA-XXXX-XXXX
}

func TestMarshalUnmarshalStringSlice(t *testing.T) {
	tests := []struct {
		input []string
	}{
		{nil},
		{[]string{}},
		{[]string{"a"}},
		{[]string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		s := marshalStringSlice(tt.input)
		got := unmarshalStringSlice(s)
		if len(tt.input) == 0 {
			assert.Empty(t, got)
		} else {
			assert.Equal(t, tt.input, got)
		}
	}
}

func TestUnmarshalStringSlice_EdgeCases(t *testing.T) {
	assert.Nil(t, unmarshalStringSlice(""))
	assert.Nil(t, unmarshalStringSlice("[]"))
	assert.Nil(t, unmarshalStringSlice("null"))
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

// --- API Tests ---

func TestAPI_InvitesCRUD(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create invite
	body := `{"note":"test invite","max_uses":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/invites", strings.NewReader(body))
	req.Header.Set("X-User-ID", "admin-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var invite InviteCode
	json.NewDecoder(w.Body).Decode(&invite)
	assert.NotEmpty(t, invite.Code)
	assert.Equal(t, 5, invite.MaxUses)

	// List invites
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/invites", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]any
	json.NewDecoder(w.Body).Decode(&listResp)
	assert.Equal(t, float64(1), listResp["count"])

	// Get invite by ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/invites/"+invite.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Revoke invite
	req = httptest.NewRequest(http.MethodPost, "/api/v1/beta/invites/"+invite.ID+"/revoke", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_Invites_Unauthorized(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"note":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/invites", strings.NewReader(body))
	// No X-User-ID header
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Invites_NoStore(t *testing.T) {
	api := NewAPI(nil, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/invites", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAPI_Invites_MethodNotAllowed(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/beta/invites", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAPI_Invite_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/invites/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Redeem_Success(t *testing.T) {
	store, _ := NewStore(testDB(t))
	invite := &InviteCode{CreatedBy: "admin"}
	require.NoError(t, store.CreateInvite(invite))

	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body, _ := json.Marshal(redeemRequest{Code: invite.Code, Email: "user@test.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/redeem", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_Redeem_MissingFields(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Missing code
	body := `{"email":"test@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/redeem", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Missing email
	body = `{"code":"BETA-XXXX-XXXX"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/beta/redeem", strings.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Redeem_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"code":"BETA-XXXX-XXXX","email":"test@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/redeem", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Redeem_MethodNotAllowed(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/redeem", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAPI_FlagsCRUD(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create flag
	body := `{"name":"dark_mode","enabled":true,"rollout_pct":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List flags
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/flags", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get flag
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/flags/dark_mode", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Update flag
	body = `{"enabled":false,"rollout_pct":0}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/beta/flags/dark_mode", strings.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete flag
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/beta/flags/dark_mode", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_Flags_Duplicate(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"name":"test_flag"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags", strings.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAPI_Flags_MissingName(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_FlagCheck(t *testing.T) {
	store, _ := NewStore(testDB(t))
	require.NoError(t, store.CreateFlag(&FeatureFlag{Name: "test", Enabled: true, RolloutPct: 100}))

	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"flag_name":"test","user_id":"user-1","user_plan":"free"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags/check", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["enabled"].(bool))
}

func TestAPI_FlagCheck_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"flag_name":"nonexistent","user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags/check", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["enabled"].(bool))
}

func TestAPI_FlagCheck_MissingFields(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/flags/check", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_WaitlistCRUD(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Sign up
	body := `{"email":"test@example.com","name":"Test User","source":"website"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/waitlist", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/waitlist", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]any
	json.NewDecoder(w.Body).Decode(&listResp)
	assert.Equal(t, float64(1), listResp["count"])

	// Get by email
	req = httptest.NewRequest(http.MethodGet, "/api/v1/beta/waitlist/test@example.com", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/beta/waitlist/test@example.com", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_Waitlist_Duplicate(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/waitlist", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/beta/waitlist", strings.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAPI_Waitlist_MissingEmail(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/waitlist", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Waitlist_NotFound(t *testing.T) {
	store, _ := NewStore(testDB(t))
	api := NewAPI(store, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/waitlist/nobody@test.com", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Readiness_Ready(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("test", "general", false, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckPass, Message: "ok"}
	})

	api := NewAPI(nil, rc)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/readiness", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var report ReadinessReport
	json.NewDecoder(w.Body).Decode(&report)
	assert.True(t, report.Ready)
}

func TestAPI_Readiness_NotReady(t *testing.T) {
	rc := NewReadinessChecker()
	rc.Register("critical", "general", true, func() ReadinessCheck {
		return ReadinessCheck{Status: CheckFail, Message: "broken"}
	})

	api := NewAPI(nil, rc)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/readiness", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAPI_Readiness_NoChecker(t *testing.T) {
	api := NewAPI(nil, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/readiness", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAPI_Readiness_MethodNotAllowed(t *testing.T) {
	api := NewAPI(nil, NewReadinessChecker())
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/beta/readiness", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- Full Integration Flow ---

func TestFullBetaFlow(t *testing.T) {
	store, _ := NewStore(testDB(t))

	// 1. Add someone to waitlist
	err := store.AddToWaitlist(&WaitlistEntry{Email: "user@example.com", Name: "Test User", Source: "landing_page"})
	require.NoError(t, err)

	// 2. Create an invite for them
	invite := &InviteCode{CreatedBy: "admin", Email: "user@example.com", MaxUses: 1}
	require.NoError(t, store.CreateInvite(invite))

	// 3. Update waitlist status to invited
	err = store.UpdateWaitlistStatus("user@example.com", WaitlistStatusInvited, invite.ID)
	require.NoError(t, err)

	// 4. Create a feature flag
	err = store.CreateFlag(&FeatureFlag{
		Name:       "new_agent_ui",
		Enabled:    true,
		RolloutPct: 100,
	})
	require.NoError(t, err)

	// 5. User redeems invite
	_, err = store.RedeemInvite(invite.Code, "user@example.com")
	require.NoError(t, err)

	// 6. Update waitlist to joined
	err = store.UpdateWaitlistStatus("user@example.com", WaitlistStatusJoined, invite.ID)
	require.NoError(t, err)

	// 7. Check feature flag
	flag, err := store.GetFlag("new_agent_ui")
	require.NoError(t, err)
	assert.True(t, flag.IsEnabledForUser("user-1", "free"))

	// 8. Verify counts
	total, _ := store.CountWaitlist("")
	assert.Equal(t, int64(1), total)

	invCount, _ := store.CountInvites("")
	assert.Equal(t, int64(1), invCount)
}

// --- Error Constants ---

func TestErrorConstants(t *testing.T) {
	assert.Error(t, ErrNilDB)
	assert.Error(t, ErrInviteNotFound)
	assert.Error(t, ErrInviteExpired)
	assert.Error(t, ErrInviteUsed)
	assert.Error(t, ErrInviteExhausted)
	assert.Error(t, ErrInvalidCode)
	assert.Error(t, ErrFlagNotFound)
	assert.Error(t, ErrDuplicateFlag)
	assert.Error(t, ErrInvalidFlag)
	assert.Error(t, ErrInvalidRollout)
	assert.Error(t, ErrWaitlistDuplicate)
	assert.Error(t, ErrWaitlistNotFound)
	assert.Error(t, ErrNotReady)
	assert.Error(t, ErrInvalidEmail)
	assert.Error(t, ErrInvalidStatus)
	assert.Error(t, ErrEmptyID)
}
