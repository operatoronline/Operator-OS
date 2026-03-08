package gdpr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	t.Cleanup(func() { db.Close() })
	return db
}

// ---------- Mock data source ----------

type mockSource struct {
	name       string
	exportData interface{}
	exportErr  error
	eraseCount int64
	eraseErr   error
}

func (m *mockSource) Name() string { return m.name }
func (m *mockSource) Export(_ context.Context, _ string) (interface{}, int, error) {
	if m.exportErr != nil {
		return nil, 0, m.exportErr
	}
	if data, ok := m.exportData.([]interface{}); ok {
		return data, len(data), nil
	}
	return m.exportData, 1, nil
}
func (m *mockSource) Erase(_ context.Context, _ string) (int64, error) {
	return m.eraseCount, m.eraseErr
}

// ---------- ValidRequestType / ValidRequestStatus ----------

func TestValidRequestType(t *testing.T) {
	assert.True(t, ValidRequestType(TypeExport))
	assert.True(t, ValidRequestType(TypeErasure))
	assert.False(t, ValidRequestType("unknown"))
	assert.False(t, ValidRequestType(""))
}

func TestValidRequestStatus(t *testing.T) {
	for _, s := range []string{StatusPending, StatusProcessing, StatusCompleted, StatusFailed, StatusCanceled} {
		assert.True(t, ValidRequestStatus(s), s)
	}
	assert.False(t, ValidRequestStatus("unknown"))
	assert.False(t, ValidRequestStatus(""))
}

// ---------- RetentionPolicy ----------

func TestDefaultRetentionPolicy(t *testing.T) {
	p := DefaultRetentionPolicy()
	assert.Equal(t, 365*24*time.Hour, p.AuditLogRetention)
	assert.Equal(t, 180*24*time.Hour, p.UsageDataRetention)
	assert.Equal(t, 90*24*time.Hour, p.SessionRetention)
	assert.Equal(t, 30*24*time.Hour, p.DeletedUserRetention)
}

// ---------- RequestStore ----------

func TestNewRequestStore_NilDB(t *testing.T) {
	_, err := NewRequestStore(nil)
	assert.ErrorIs(t, err, ErrNilDB)
}

func TestNewRequestStore_OK(t *testing.T) {
	db := openTestDB(t)
	store, err := NewRequestStore(db)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestRequestStore_Create(t *testing.T) {
	db := openTestDB(t)
	store, err := NewRequestStore(db)
	require.NoError(t, err)

	req := &DataSubjectRequest{
		UserID:      "user-1",
		Type:        TypeExport,
		RequestedBy: "user-1",
	}
	err = store.Create(req)
	require.NoError(t, err)
	assert.NotEmpty(t, req.ID)
	assert.Equal(t, StatusPending, req.Status)
	assert.False(t, req.CreatedAt.IsZero())
}

func TestRequestStore_Create_NilRequest(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.Error(t, store.Create(nil))
}

func TestRequestStore_Create_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.Create(&DataSubjectRequest{Type: TypeExport}), ErrUserIDRequired)
}

func TestRequestStore_Create_InvalidType(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: "bad"}), ErrInvalidType)
}

func TestRequestStore_Create_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	err := store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport, Status: "bad"})
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestRequestStore_Create_CustomID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	req := &DataSubjectRequest{
		ID:     "custom-id",
		UserID: "user-1",
		Type:   TypeErasure,
	}
	require.NoError(t, store.Create(req))
	assert.Equal(t, "custom-id", req.ID)
}

func TestRequestStore_GetByID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport, RequestedBy: "admin-1", Notes: "test note"}
	require.NoError(t, store.Create(req))

	got, err := store.GetByID(req.ID)
	require.NoError(t, err)
	assert.Equal(t, req.ID, got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, TypeExport, got.Type)
	assert.Equal(t, StatusPending, got.Status)
	assert.Equal(t, "admin-1", got.RequestedBy)
	assert.Equal(t, "test note", got.Notes)
}

func TestRequestStore_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	_, err := store.GetByID("nonexistent")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

func TestRequestStore_GetByID_Empty(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	_, err := store.GetByID("")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

func TestRequestStore_ListByUser(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-1", Type: TypeErasure}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-2", Type: TypeExport}))

	list, err := store.ListByUser("user-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestRequestStore_ListByUser_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	_, err := store.ListByUser("")
	assert.ErrorIs(t, err, ErrUserIDRequired)
}

func TestRequestStore_ListByStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-2", Type: TypeErasure}))

	list, err := store.ListByStatus(StatusPending)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestRequestStore_ListByStatus_Invalid(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	_, err := store.ListByStatus("bad")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestRequestStore_UpdateStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	err := store.UpdateStatus(req.ID, StatusProcessing, "")
	require.NoError(t, err)

	got, _ := store.GetByID(req.ID)
	assert.Equal(t, StatusProcessing, got.Status)
}

func TestRequestStore_UpdateStatus_Completed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	err := store.UpdateStatus(req.ID, StatusCompleted, "")
	require.NoError(t, err)

	got, _ := store.GetByID(req.ID)
	assert.Equal(t, StatusCompleted, got.Status)
	assert.False(t, got.CompletedAt.IsZero())
}

func TestRequestStore_UpdateStatus_Failed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	err := store.UpdateStatus(req.ID, StatusFailed, "something broke")
	require.NoError(t, err)

	got, _ := store.GetByID(req.ID)
	assert.Equal(t, StatusFailed, got.Status)
	assert.Equal(t, "something broke", got.ErrorMsg)
	assert.False(t, got.CompletedAt.IsZero())
}

func TestRequestStore_UpdateStatus_NotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.UpdateStatus("nope", StatusCompleted, ""), ErrRequestNotFound)
}

func TestRequestStore_UpdateStatus_EmptyID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.UpdateStatus("", StatusCompleted, ""), ErrRequestNotFound)
}

func TestRequestStore_UpdateStatus_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.UpdateStatus("some-id", "bad", ""), ErrInvalidStatus)
}

func TestRequestStore_SetResult(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	err := store.SetResult(req.ID, `{"data":"exported"}`)
	require.NoError(t, err)

	got, _ := store.GetByID(req.ID)
	assert.Equal(t, StatusCompleted, got.Status)
	assert.Equal(t, `{"data":"exported"}`, got.ResultData)
	assert.False(t, got.CompletedAt.IsZero())
}

func TestRequestStore_SetResult_NotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.SetResult("nope", "data"), ErrRequestNotFound)
}

func TestRequestStore_SetResult_EmptyID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.ErrorIs(t, store.SetResult("", "data"), ErrRequestNotFound)
}

func TestRequestStore_Count(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u2", Type: TypeErasure}))

	total, err := store.Count("")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	pending, err := store.Count(StatusPending)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pending)

	completed, err := store.Count(StatusCompleted)
	require.NoError(t, err)
	assert.Equal(t, int64(0), completed)
}

func TestRequestStore_Count_InvalidStatus(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	_, err := store.Count("bad")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestRequestStore_DeleteBefore(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{UserID: "u1", Type: TypeExport}
	require.NoError(t, store.Create(req))
	require.NoError(t, store.SetResult(req.ID, "data"))

	// Delete before far future — should remove it
	n, err := store.DeleteBefore(time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	total, _ := store.Count("")
	assert.Equal(t, int64(0), total)
}

func TestRequestStore_DeleteBefore_NoPending(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	// Pending request should NOT be deleted (no completed_at)
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport}))

	n, err := store.DeleteBefore(time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestRequestStore_Close(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	assert.NoError(t, store.Close())
}

// ---------- Service ----------

func TestNewService_NilStore(t *testing.T) {
	_, err := NewService(ServiceConfig{})
	assert.Error(t, err)
}

func TestNewService_OK(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, err := NewService(ServiceConfig{RequestStore: store})
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewService_DefaultRetention(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	assert.Equal(t, DefaultRetentionPolicy(), svc.retention)
}

func TestNewService_CustomRetention(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	custom := RetentionPolicy{AuditLogRetention: 30 * 24 * time.Hour}
	svc, _ := NewService(ServiceConfig{RequestStore: store, Retention: custom})
	assert.Equal(t, custom, svc.retention)
}

// ---------- Export ----------

func TestService_ExportUserData(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	src := &mockSource{
		name:       "accounts",
		exportData: []interface{}{"item1", "item2"},
	}

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources:      []UserDataSource{src},
	})

	export, err := svc.ExportUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", export.UserID)
	assert.Len(t, export.Sections, 1)
	assert.Equal(t, "accounts", export.Sections[0].Name)
	assert.Equal(t, 2, export.Sections[0].ItemCount)
}

func TestService_ExportUserData_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	_, err := svc.ExportUserData(context.Background(), "", "admin")
	assert.ErrorIs(t, err, ErrUserIDRequired)
}

func TestService_ExportUserData_SourceError(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	src := &mockSource{
		name:      "broken",
		exportErr: errors.New("db down"),
	}

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources:      []UserDataSource{src},
	})

	export, err := svc.ExportUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err) // partial export is not an error
	assert.Len(t, export.Sections, 1)
	// Error captured in section data
	data, ok := export.Sections[0].Data.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "db down", data["error"])
}

func TestService_ExportUserData_MultipleSources(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources: []UserDataSource{
			&mockSource{name: "accounts", exportData: "user data"},
			&mockSource{name: "sessions", exportData: []interface{}{"s1", "s2", "s3"}},
			&mockSource{name: "billing", exportData: "billing data"},
		},
	})

	export, err := svc.ExportUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Len(t, export.Sections, 3)
	assert.Equal(t, "accounts", export.Sections[0].Name)
	assert.Equal(t, "sessions", export.Sections[1].Name)
	assert.Equal(t, 3, export.Sections[1].ItemCount)
	assert.Equal(t, "billing", export.Sections[2].Name)
}

func TestService_ExportUserData_CreatesRequest(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	_, err := svc.ExportUserData(context.Background(), "user-1", "admin")
	require.NoError(t, err)

	requests, err := store.ListByUser("user-1")
	require.NoError(t, err)
	assert.Len(t, requests, 1)
	assert.Equal(t, TypeExport, requests[0].Type)
	assert.Equal(t, StatusCompleted, requests[0].Status)
}

func TestService_ExportUserData_NoSources(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	export, err := svc.ExportUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Empty(t, export.Sections)
}

// ---------- Erase ----------

func TestService_EraseUserData(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources: []UserDataSource{
			&mockSource{name: "accounts", eraseCount: 1},
			&mockSource{name: "sessions", eraseCount: 5},
		},
	})

	report, err := svc.EraseUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", report.UserID)
	assert.Len(t, report.Sections, 2)
	assert.Equal(t, int64(1), report.Sections[0].ItemsDeleted)
	assert.Equal(t, int64(5), report.Sections[1].ItemsDeleted)
	assert.Equal(t, int64(6), report.TotalItems)
}

func TestService_EraseUserData_EmptyUserID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	_, err := svc.EraseUserData(context.Background(), "", "admin")
	assert.ErrorIs(t, err, ErrUserIDRequired)
}

func TestService_EraseUserData_PartialFailure(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources: []UserDataSource{
			&mockSource{name: "accounts", eraseCount: 1},
			&mockSource{name: "broken", eraseErr: errors.New("access denied")},
		},
	})

	report, err := svc.EraseUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err) // partial failure is not a top-level error
	assert.Len(t, report.Sections, 2)
	assert.Empty(t, report.Sections[0].Error)
	assert.Equal(t, "access denied", report.Sections[1].Error)

	// Request should be marked as failed
	requests, _ := store.ListByUser("user-1")
	assert.Equal(t, StatusFailed, requests[0].Status)
}

func TestService_EraseUserData_AllSuccess(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources:      []UserDataSource{&mockSource{name: "data", eraseCount: 3}},
	})

	_, err := svc.EraseUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)

	requests, _ := store.ListByUser("user-1")
	assert.Equal(t, StatusCompleted, requests[0].Status)
}

// ---------- Retention ----------

func TestService_EnforceRetention(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	called := false
	enforcer := NewRetentionEnforcer([]RetentionRule{
		{
			Name:      "audit_logs",
			Retention: 365 * 24 * time.Hour,
			Cleanup: func(_ context.Context, before time.Time) (int64, error) {
				called = true
				return 42, nil
			},
		},
	})

	report, err := svc.EnforceRetention(context.Background(), enforcer)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Len(t, report.Sections, 1)
	assert.Equal(t, int64(42), report.Sections[0].ItemsRemoved)
	assert.Equal(t, "audit_logs", report.Sections[0].Name)
	assert.Equal(t, int64(42), report.TotalItems)
}

func TestService_EnforceRetention_Error(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	enforcer := NewRetentionEnforcer([]RetentionRule{
		{
			Name:      "broken",
			Retention: time.Hour,
			Cleanup: func(_ context.Context, _ time.Time) (int64, error) {
				return 0, errors.New("cleanup failed")
			},
		},
	})

	report, err := svc.EnforceRetention(context.Background(), enforcer)
	require.NoError(t, err) // errors in individual rules don't fail the whole operation
	assert.Equal(t, "cleanup failed", report.Sections[0].Error)
}

func TestService_EnforceRetention_NilRules(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	enforcer := NewRetentionEnforcer(nil)
	report, err := svc.EnforceRetention(context.Background(), enforcer)
	require.NoError(t, err)
	assert.Empty(t, report.Sections)
	assert.Equal(t, int64(0), report.TotalItems)
}

func TestService_EnforceRetention_MultipleRules(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	enforcer := NewRetentionEnforcer([]RetentionRule{
		{Name: "audit", Retention: 365 * 24 * time.Hour, Cleanup: func(_ context.Context, _ time.Time) (int64, error) { return 10, nil }},
		{Name: "usage", Retention: 180 * 24 * time.Hour, Cleanup: func(_ context.Context, _ time.Time) (int64, error) { return 20, nil }},
		{Name: "sessions", Retention: 90 * 24 * time.Hour, Cleanup: func(_ context.Context, _ time.Time) (int64, error) { return 30, nil }},
	})

	report, err := svc.EnforceRetention(context.Background(), enforcer)
	require.NoError(t, err)
	assert.Len(t, report.Sections, 3)
	assert.Equal(t, int64(60), report.TotalItems)
}

// ---------- Request management ----------

func TestService_GetRequest(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	req := &DataSubjectRequest{UserID: "u1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	got, err := svc.GetRequest(req.ID)
	require.NoError(t, err)
	assert.Equal(t, req.ID, got.ID)
}

func TestService_ListUserRequests(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeErasure}))

	list, err := svc.ListUserRequests("u1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestService_ListPendingRequests(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport}))

	list, err := svc.ListPendingRequests()
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestService_CancelRequest(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	req := &DataSubjectRequest{UserID: "u1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	err := svc.CancelRequest(req.ID)
	require.NoError(t, err)

	got, _ := svc.GetRequest(req.ID)
	assert.Equal(t, StatusCanceled, got.Status)
}

func TestService_CancelRequest_AlreadyProcessed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	req := &DataSubjectRequest{UserID: "u1", Type: TypeExport}
	require.NoError(t, store.Create(req))
	require.NoError(t, store.UpdateStatus(req.ID, StatusCompleted, ""))

	err := svc.CancelRequest(req.ID)
	assert.ErrorIs(t, err, ErrAlreadyProcessed)
}

func TestService_CancelRequest_NotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})

	err := svc.CancelRequest("nonexistent")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

// ---------- RetentionEnforcer ----------

func TestRetentionEnforcer_NilReceiver(t *testing.T) {
	var e *DefaultRetentionEnforcer
	assert.Nil(t, e.Rules())
}

func TestRetentionEnforcer_Empty(t *testing.T) {
	e := NewRetentionEnforcer(nil)
	assert.Nil(t, e.Rules())
}

// ---------- DataExport JSON ----------

func TestDataExport_JSON(t *testing.T) {
	export := DataExport{
		UserID:     "user-1",
		ExportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Sections: []ExportSection{
			{Name: "accounts", ItemCount: 1, Data: map[string]string{"email": "test@example.com"}},
		},
	}
	b, err := json.Marshal(export)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"user_id":"user-1"`)
	assert.Contains(t, string(b), `"accounts"`)
}

func TestErasureReport_JSON(t *testing.T) {
	report := ErasureReport{
		UserID:   "user-1",
		ErasedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Sections: []ErasureSection{
			{Name: "sessions", ItemsDeleted: 5},
		},
		TotalItems: 5,
	}
	b, err := json.Marshal(report)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"total_items":5`)
}

// ---------- Multi-user isolation ----------

func TestRequestStore_MultiUserIsolation(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u2", Type: TypeErasure}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "u2", Type: TypeExport}))

	u1, _ := store.ListByUser("u1")
	u2, _ := store.ListByUser("u2")
	assert.Len(t, u1, 1)
	assert.Len(t, u2, 2)
}

// ---------- API tests ----------

func makeAuthRequest(method, path, body, userID string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if userID != "" {
		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		r = r.WithContext(ctx)
	}
	return r
}

func TestAPI_Export_Unauthorized(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleExport(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/export", "", ""))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Export_MethodNotAllowed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleExport(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/export", "", "user-1"))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAPI_Export_NoService(t *testing.T) {
	api := NewAPI(nil)
	w := httptest.NewRecorder()
	api.handleExport(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/export", "", "user-1"))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAPI_Export_Success(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources:      []UserDataSource{&mockSource{name: "test", exportData: "data"}},
	})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleExport(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/export", "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)

	var export DataExport
	require.NoError(t, json.NewDecoder(w.Body).Decode(&export))
	assert.Equal(t, "user-1", export.UserID)
	assert.Len(t, export.Sections, 1)
}

func TestAPI_Erase_NoConfirm(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleErase(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/erase", `{}`, "user-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "confirmation required")
}

func TestAPI_Erase_Success(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources:      []UserDataSource{&mockSource{name: "test", eraseCount: 3}},
	})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleErase(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/erase", `{"confirm":true}`, "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)

	var report ErasureReport
	require.NoError(t, json.NewDecoder(w.Body).Decode(&report))
	assert.Equal(t, int64(3), report.TotalItems)
}

func TestAPI_Erase_Unauthorized(t *testing.T) {
	api := NewAPI(nil)
	w := httptest.NewRecorder()
	api.handleErase(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/erase", `{"confirm":true}`, ""))
	// nil service should be checked first, but auth is checked after method
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAPI_Erase_MethodNotAllowed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleErase(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/erase", "", "user-1"))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAPI_Requests_List(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	// Create some requests
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-1", Type: TypeExport}))
	require.NoError(t, store.Create(&DataSubjectRequest{UserID: "user-1", Type: TypeErasure}))

	w := httptest.NewRecorder()
	api.handleRequests(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests", "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Requests []*DataSubjectRequest `json:"requests"`
		Count    int                   `json:"count"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count)
}

func TestAPI_Requests_Unauthorized(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRequests(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests", "", ""))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Requests_Empty(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRequests(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests", "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":0`)
}

func TestAPI_RequestByID_Get(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests/"+req.ID, "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)

	var got DataSubjectRequest
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, req.ID, got.ID)
}

func TestAPI_RequestByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests/nope", "", "user-1"))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_RequestByID_WrongUser(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests/"+req.ID, "", "user-2"))
	assert.Equal(t, http.StatusNotFound, w.Code) // should not reveal existence
}

func TestAPI_RequestByID_Cancel(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodDelete, "/api/v1/gdpr/requests/"+req.ID, "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "canceled")
}

func TestAPI_RequestByID_CancelAlreadyProcessed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	req := &DataSubjectRequest{UserID: "user-1", Type: TypeExport}
	require.NoError(t, store.Create(req))
	require.NoError(t, store.SetResult(req.ID, "done"))

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodDelete, "/api/v1/gdpr/requests/"+req.ID, "", "user-1"))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAPI_RequestByID_EmptyID(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRequestByID(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/requests/", "", "user-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Retention_Get(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRetention(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/retention", "", "user-1"))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]int
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 365, resp["audit_log_days"])
	assert.Equal(t, 180, resp["usage_data_days"])
	assert.Equal(t, 90, resp["session_days"])
	assert.Equal(t, 30, resp["deleted_user_days"])
}

func TestAPI_Retention_Unauthorized(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRetention(w, makeAuthRequest(http.MethodGet, "/api/v1/gdpr/retention", "", ""))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Retention_MethodNotAllowed(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	w := httptest.NewRecorder()
	api.handleRetention(w, makeAuthRequest(http.MethodPost, "/api/v1/gdpr/retention", "", "user-1"))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAPI_RegisterRoutes(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)
	svc, _ := NewService(ServiceConfig{RequestStore: store})
	api := NewAPI(svc)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Verify routes are registered by making requests
	for _, path := range []string{
		"/api/v1/gdpr/export",
		"/api/v1/gdpr/erase",
		"/api/v1/gdpr/requests",
		"/api/v1/gdpr/retention",
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, makeAuthRequest(http.MethodGet, path, "", "user-1"))
		// Should not be 404 (method might be wrong, but route should exist)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s not registered", path)
	}
}

// ---------- generateID ----------

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32) // 16 bytes = 32 hex chars
}

// ---------- Persistence ----------

func TestRequestStore_Persistence(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	req := &DataSubjectRequest{
		UserID:      "user-1",
		Type:        TypeExport,
		RequestedBy: "admin",
		Notes:       "test note",
	}
	require.NoError(t, store.Create(req))
	require.NoError(t, store.UpdateStatus(req.ID, StatusCompleted, ""))

	// Open a new store on the same DB
	store2, err := NewRequestStore(db)
	require.NoError(t, err)

	got, err := store2.GetByID(req.ID)
	require.NoError(t, err)
	assert.Equal(t, req.ID, got.ID)
	assert.Equal(t, StatusCompleted, got.Status)
	assert.Equal(t, "test note", got.Notes)
}

// ---------- Full lifecycle ----------

func TestFullLifecycle_ExportThenErase(t *testing.T) {
	db := openTestDB(t)
	store, _ := NewRequestStore(db)

	userData := map[string]string{"email": "test@example.com", "name": "Test User"}

	svc, _ := NewService(ServiceConfig{
		RequestStore: store,
		Sources: []UserDataSource{
			&mockSource{
				name:       "accounts",
				exportData: userData,
				eraseCount: 1,
			},
			&mockSource{
				name:       "sessions",
				exportData: []interface{}{"s1", "s2"},
				eraseCount: 2,
			},
		},
	})

	// Step 1: Export
	export, err := svc.ExportUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", export.UserID)
	assert.Len(t, export.Sections, 2)

	// Step 2: Erase
	report, err := svc.EraseUserData(context.Background(), "user-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), report.TotalItems)

	// Step 3: Verify DSR records
	requests, err := svc.ListUserRequests("user-1")
	require.NoError(t, err)
	assert.Len(t, requests, 2)
	// Most recent first
	assert.Equal(t, TypeErasure, requests[0].Type)
	assert.Equal(t, TypeExport, requests[1].Type)
}

// ---------- UserDataSource interface compliance ----------

func TestUserDataSource_Interface(t *testing.T) {
	// Verify mock implements the interface
	var _ UserDataSource = &mockSource{}
}

// ---------- Error constants ----------

func TestErrorConstants(t *testing.T) {
	errors := []error{
		ErrNilDB, ErrUserIDRequired, ErrRequestNotFound,
		ErrInvalidType, ErrInvalidStatus, ErrAlreadyProcessed, ErrNilConfig,
	}
	for _, e := range errors {
		assert.NotEmpty(t, e.Error())
	}
}

// ---------- Type/status constants ----------

func TestTypeConstants(t *testing.T) {
	assert.Equal(t, "export", TypeExport)
	assert.Equal(t, "erasure", TypeErasure)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "processing", StatusProcessing)
	assert.Equal(t, "completed", StatusCompleted)
	assert.Equal(t, "failed", StatusFailed)
	assert.Equal(t, "canceled", StatusCanceled)
}

// ---------- Edge cases ----------

func TestRequestStore_ConcurrentReads(t *testing.T) {
	// Use a temp file DB for concurrent access stability under -race
	dir := t.TempDir()
	db, err := sql.Open("sqlite", dir+"/test.db?_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	store, err := NewRequestStore(db)
	require.NoError(t, err)

	// Create records sequentially first
	const n = 10
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		req := &DataSubjectRequest{
			UserID: fmt.Sprintf("user-%d", i),
			Type:   TypeExport,
		}
		require.NoError(t, store.Create(req))
		ids[i] = req.ID
	}

	// Verify creation
	total, err := store.Count("")
	require.NoError(t, err)
	require.Equal(t, int64(n), total)

	// Concurrent reads should be safe
	done := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			got, _ := store.GetByID(ids[idx])
			assert.NotNil(t, got)
			list, _ := store.ListByUser(fmt.Sprintf("user-%d", idx))
			assert.Len(t, list, 1)
			done <- true
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}
}

func TestRetentionReport_JSON(t *testing.T) {
	report := RetentionReport{
		EnforcedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Sections: []RetentionResult{
			{Name: "audit", Cutoff: "2025-01-01T00:00:00Z", ItemsRemoved: 100},
		},
		TotalItems: 100,
	}
	b, err := json.Marshal(report)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"total_items":100`)
}
