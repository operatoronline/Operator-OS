// Package gdpr provides GDPR compliance tools for Operator OS.
//
// It implements data subject rights including data export (right to access),
// data erasure (right to deletion), and configurable retention policies.
// A DataSubjectRequest model tracks the lifecycle of compliance requests.
package gdpr

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Errors returned by the GDPR package.
var (
	ErrNilDB            = errors.New("gdpr: nil database")
	ErrUserIDRequired   = errors.New("gdpr: user ID is required")
	ErrRequestNotFound  = errors.New("gdpr: request not found")
	ErrInvalidType      = errors.New("gdpr: invalid request type")
	ErrInvalidStatus    = errors.New("gdpr: invalid request status")
	ErrAlreadyProcessed = errors.New("gdpr: request already processed")
	ErrNilConfig        = errors.New("gdpr: nil config")
)

// Request type constants.
const (
	TypeExport  = "export"
	TypeErasure = "erasure"
)

// Request status constants.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCanceled   = "canceled"
)

// ValidRequestType reports whether t is a known request type.
func ValidRequestType(t string) bool {
	return t == TypeExport || t == TypeErasure
}

// ValidRequestStatus reports whether s is a known request status.
func ValidRequestStatus(s string) bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed, StatusCanceled:
		return true
	}
	return false
}

// DataSubjectRequest tracks a GDPR data subject request (export or erasure).
type DataSubjectRequest struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"`         // "export" or "erasure"
	Status      string    `json:"status"`       // pending, processing, completed, failed, canceled
	RequestedBy string    `json:"requested_by"` // user ID or admin ID
	Notes       string    `json:"notes,omitempty"`
	ResultData  string    `json:"result_data,omitempty"` // JSON export data or completion metadata
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// DataExport represents all user data collected for a GDPR data export.
type DataExport struct {
	UserID      string          `json:"user_id"`
	ExportedAt  time.Time       `json:"exported_at"`
	Sections    []ExportSection `json:"sections"`
}

// ExportSection groups exported data by category.
type ExportSection struct {
	Name      string      `json:"name"`
	ItemCount int         `json:"item_count"`
	Data      interface{} `json:"data"`
}

// ErasureReport summarizes what was deleted during a data erasure.
type ErasureReport struct {
	UserID     string           `json:"user_id"`
	ErasedAt   time.Time        `json:"erased_at"`
	Sections   []ErasureSection `json:"sections"`
	TotalItems int64            `json:"total_items"`
}

// ErasureSection records deletions for a specific data category.
type ErasureSection struct {
	Name         string `json:"name"`
	ItemsDeleted int64  `json:"items_deleted"`
	Error        string `json:"error,omitempty"`
}

// RetentionPolicy defines how long each category of data is retained.
type RetentionPolicy struct {
	// AuditLogRetention is how long audit events are kept.
	AuditLogRetention time.Duration `json:"audit_log_retention"`
	// UsageDataRetention is how long usage events are kept.
	UsageDataRetention time.Duration `json:"usage_data_retention"`
	// SessionRetention is how long inactive sessions are kept.
	SessionRetention time.Duration `json:"session_retention"`
	// DeletedUserRetention is how long after deletion we keep residual data.
	DeletedUserRetention time.Duration `json:"deleted_user_retention"`
}

// DefaultRetentionPolicy returns sensible retention defaults.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		AuditLogRetention:    365 * 24 * time.Hour,  // 1 year
		UsageDataRetention:   180 * 24 * time.Hour,  // 6 months
		SessionRetention:     90 * 24 * time.Hour,   // 3 months
		DeletedUserRetention: 30 * 24 * time.Hour,   // 30 days
	}
}

// RetentionReport summarizes what was cleaned up by retention enforcement.
type RetentionReport struct {
	EnforcedAt time.Time          `json:"enforced_at"`
	Sections   []RetentionResult  `json:"sections"`
	TotalItems int64              `json:"total_items"`
}

// RetentionResult records cleanup for one data category.
type RetentionResult struct {
	Name         string `json:"name"`
	Cutoff       string `json:"cutoff"`
	ItemsRemoved int64  `json:"items_removed"`
	Error        string `json:"error,omitempty"`
}

// ---------- Data source interfaces ----------

// UserDataSource can export and delete user data for a specific domain.
type UserDataSource interface {
	// Name returns a human-readable name for this data source.
	Name() string
	// Export returns all data for the given user as a JSON-serializable value.
	Export(ctx context.Context, userID string) (interface{}, int, error)
	// Erase deletes all data for the given user, returning the count of items deleted.
	Erase(ctx context.Context, userID string) (int64, error)
}

// ---------- Request store ----------

const createDSRTableSQL = `
CREATE TABLE IF NOT EXISTS data_subject_requests (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL,
	type         TEXT NOT NULL,
	status       TEXT NOT NULL DEFAULT 'pending',
	requested_by TEXT NOT NULL DEFAULT '',
	notes        TEXT NOT NULL DEFAULT '',
	result_data  TEXT NOT NULL DEFAULT '',
	error_msg    TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL,
	completed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_dsr_user_id ON data_subject_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_dsr_status ON data_subject_requests(status);
CREATE INDEX IF NOT EXISTS idx_dsr_type ON data_subject_requests(type);
`

// RequestStore persists and queries data subject requests.
type RequestStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewRequestStore creates a new RequestStore, initializing the schema.
func NewRequestStore(db *sql.DB) (*RequestStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if _, err := db.Exec(createDSRTableSQL); err != nil {
		return nil, fmt.Errorf("gdpr: create table: %w", err)
	}
	return &RequestStore{db: db}, nil
}

// Create inserts a new data subject request.
func (s *RequestStore) Create(req *DataSubjectRequest) error {
	if req == nil {
		return errors.New("gdpr: nil request")
	}
	if req.UserID == "" {
		return ErrUserIDRequired
	}
	if !ValidRequestType(req.Type) {
		return ErrInvalidType
	}

	if req.ID == "" {
		req.ID = generateID()
	}
	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	req.UpdatedAt = now
	if req.Status == "" {
		req.Status = StatusPending
	}
	if !ValidRequestStatus(req.Status) {
		return ErrInvalidStatus
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`INSERT INTO data_subject_requests 
		(id, user_id, type, status, requested_by, notes, result_data, error_msg, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.UserID, req.Type, req.Status, req.RequestedBy,
		req.Notes, req.ResultData, req.ErrorMsg,
		req.CreatedAt.Format(time.RFC3339Nano),
		req.UpdatedAt.Format(time.RFC3339Nano),
		"",
	)
	return err
}

// GetByID returns the request with the given ID.
func (s *RequestStore) GetByID(id string) (*DataSubjectRequest, error) {
	if id == "" {
		return nil, ErrRequestNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanOne(`SELECT id, user_id, type, status, requested_by, notes, result_data, error_msg, created_at, updated_at, completed_at 
		FROM data_subject_requests WHERE id = ?`, id)
}

// ListByUser returns all requests for a user, ordered by created_at DESC.
func (s *RequestStore) ListByUser(userID string) ([]*DataSubjectRequest, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanMany(`SELECT id, user_id, type, status, requested_by, notes, result_data, error_msg, created_at, updated_at, completed_at 
		FROM data_subject_requests WHERE user_id = ? ORDER BY created_at DESC`, userID)
}

// ListByStatus returns all requests with the given status.
func (s *RequestStore) ListByStatus(status string) ([]*DataSubjectRequest, error) {
	if !ValidRequestStatus(status) {
		return nil, ErrInvalidStatus
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanMany(`SELECT id, user_id, type, status, requested_by, notes, result_data, error_msg, created_at, updated_at, completed_at 
		FROM data_subject_requests WHERE status = ? ORDER BY created_at ASC`, status)
}

// UpdateStatus changes the status of a request.
func (s *RequestStore) UpdateStatus(id string, status string, errorMsg string) error {
	if id == "" {
		return ErrRequestNotFound
	}
	if !ValidRequestStatus(status) {
		return ErrInvalidStatus
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	completedAt := ""
	if status == StatusCompleted || status == StatusFailed {
		completedAt = now
	}

	res, err := s.db.Exec(`UPDATE data_subject_requests 
		SET status = ?, error_msg = ?, updated_at = ?, completed_at = CASE WHEN ? != '' THEN ? ELSE completed_at END
		WHERE id = ?`,
		status, errorMsg, now, completedAt, completedAt, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// SetResult sets the result data and marks the request as completed.
func (s *RequestStore) SetResult(id string, resultData string) error {
	if id == "" {
		return ErrRequestNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`UPDATE data_subject_requests 
		SET status = ?, result_data = ?, updated_at = ?, completed_at = ?
		WHERE id = ?`,
		StatusCompleted, resultData, now, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// Count returns the total number of requests matching optional status filter.
func (s *RequestStore) Count(status string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	if status != "" {
		if !ValidRequestStatus(status) {
			return 0, ErrInvalidStatus
		}
		err := s.db.QueryRow(`SELECT COUNT(*) FROM data_subject_requests WHERE status = ?`, status).Scan(&count)
		return count, err
	}
	err := s.db.QueryRow(`SELECT COUNT(*) FROM data_subject_requests`).Scan(&count)
	return count, err
}

// DeleteBefore removes requests completed before the given time.
func (s *RequestStore) DeleteBefore(before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`DELETE FROM data_subject_requests WHERE completed_at != '' AND completed_at < ?`,
		before.Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close is a no-op (the caller owns the DB connection).
func (s *RequestStore) Close() error {
	return nil
}

func (s *RequestStore) scanOne(query string, args ...interface{}) (*DataSubjectRequest, error) {
	row := s.db.QueryRow(query, args...)
	req, err := scanDSR(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	return req, err
}

func (s *RequestStore) scanMany(query string, args ...interface{}) ([]*DataSubjectRequest, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*DataSubjectRequest
	for rows.Next() {
		var r DataSubjectRequest
		var createdAt, updatedAt, completedAt string
		err := rows.Scan(&r.ID, &r.UserID, &r.Type, &r.Status, &r.RequestedBy,
			&r.Notes, &r.ResultData, &r.ErrorMsg,
			&createdAt, &updatedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		if completedAt != "" {
			r.CompletedAt, _ = time.Parse(time.RFC3339Nano, completedAt)
		}
		results = append(results, &r)
	}
	return results, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanDSR(s scanner) (*DataSubjectRequest, error) {
	var r DataSubjectRequest
	var createdAt, updatedAt, completedAt string
	err := s.Scan(&r.ID, &r.UserID, &r.Type, &r.Status, &r.RequestedBy,
		&r.Notes, &r.ResultData, &r.ErrorMsg,
		&createdAt, &updatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if completedAt != "" {
		r.CompletedAt, _ = time.Parse(time.RFC3339Nano, completedAt)
	}
	return &r, nil
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("dsr-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ---------- Service ----------

// ServiceConfig configures the GDPR compliance service.
type ServiceConfig struct {
	RequestStore *RequestStore
	Sources      []UserDataSource
	Retention    RetentionPolicy
}

// Service orchestrates GDPR compliance operations.
type Service struct {
	store     *RequestStore
	sources   []UserDataSource
	retention RetentionPolicy
	mu        sync.Mutex
}

// NewService creates a new GDPR compliance service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.RequestStore == nil {
		return nil, errors.New("gdpr: request store is required")
	}
	retention := cfg.Retention
	if retention == (RetentionPolicy{}) {
		retention = DefaultRetentionPolicy()
	}
	return &Service{
		store:     cfg.RequestStore,
		sources:   cfg.Sources,
		retention: retention,
	}, nil
}

// ExportUserData collects all user data from registered sources.
func (s *Service) ExportUserData(ctx context.Context, userID, requestedBy string) (*DataExport, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	// Create a DSR record
	req := &DataSubjectRequest{
		UserID:      userID,
		Type:        TypeExport,
		Status:      StatusProcessing,
		RequestedBy: requestedBy,
	}
	if err := s.store.Create(req); err != nil {
		return nil, fmt.Errorf("gdpr: create export request: %w", err)
	}

	export := &DataExport{
		UserID:     userID,
		ExportedAt: time.Now().UTC(),
	}

	for _, src := range s.sources {
		data, count, err := src.Export(ctx, userID)
		if err != nil {
			// Log the error in the section but continue with other sources
			export.Sections = append(export.Sections, ExportSection{
				Name:      src.Name(),
				ItemCount: 0,
				Data:      map[string]string{"error": err.Error()},
			})
			continue
		}
		export.Sections = append(export.Sections, ExportSection{
			Name:      src.Name(),
			ItemCount: count,
			Data:      data,
		})
	}

	// Store result
	resultJSON, err := json.Marshal(export)
	if err != nil {
		_ = s.store.UpdateStatus(req.ID, StatusFailed, err.Error())
		return export, nil // still return partial export
	}
	_ = s.store.SetResult(req.ID, string(resultJSON))

	return export, nil
}

// EraseUserData deletes all user data from registered sources.
func (s *Service) EraseUserData(ctx context.Context, userID, requestedBy string) (*ErasureReport, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a DSR record
	req := &DataSubjectRequest{
		UserID:      userID,
		Type:        TypeErasure,
		Status:      StatusProcessing,
		RequestedBy: requestedBy,
	}
	if err := s.store.Create(req); err != nil {
		return nil, fmt.Errorf("gdpr: create erasure request: %w", err)
	}

	report := &ErasureReport{
		UserID:   userID,
		ErasedAt: time.Now().UTC(),
	}

	var totalDeleted int64
	var hasError bool

	for _, src := range s.sources {
		count, err := src.Erase(ctx, userID)
		section := ErasureSection{
			Name:         src.Name(),
			ItemsDeleted: count,
		}
		if err != nil {
			section.Error = err.Error()
			hasError = true
		}
		totalDeleted += count
		report.Sections = append(report.Sections, section)
	}

	report.TotalItems = totalDeleted

	resultJSON, _ := json.Marshal(report)
	if hasError {
		_ = s.store.UpdateStatus(req.ID, StatusFailed, "some data sources failed; see report")
	} else {
		_ = s.store.SetResult(req.ID, string(resultJSON))
	}

	return report, nil
}

// EnforceRetention applies retention policies and cleans up old data.
func (s *Service) EnforceRetention(ctx context.Context, enforcer RetentionEnforcer) (*RetentionReport, error) {
	report := &RetentionReport{
		EnforcedAt: time.Now().UTC(),
	}

	now := time.Now().UTC()

	for _, rule := range enforcer.Rules() {
		cutoff := now.Add(-rule.Retention)
		count, err := rule.Cleanup(ctx, cutoff)
		result := RetentionResult{
			Name:         rule.Name,
			Cutoff:       cutoff.Format(time.RFC3339),
			ItemsRemoved: count,
		}
		if err != nil {
			result.Error = err.Error()
		}
		report.TotalItems += count
		report.Sections = append(report.Sections, result)
	}

	return report, nil
}

// GetRequest returns a DSR by ID.
func (s *Service) GetRequest(id string) (*DataSubjectRequest, error) {
	return s.store.GetByID(id)
}

// ListUserRequests returns all DSRs for a user.
func (s *Service) ListUserRequests(userID string) ([]*DataSubjectRequest, error) {
	return s.store.ListByUser(userID)
}

// ListPendingRequests returns all pending DSRs.
func (s *Service) ListPendingRequests() ([]*DataSubjectRequest, error) {
	return s.store.ListByStatus(StatusPending)
}

// CancelRequest marks a pending request as canceled.
func (s *Service) CancelRequest(id string) error {
	req, err := s.store.GetByID(id)
	if err != nil {
		return err
	}
	if req.Status != StatusPending {
		return ErrAlreadyProcessed
	}
	return s.store.UpdateStatus(id, StatusCanceled, "")
}

// ---------- Retention enforcer ----------

// RetentionRule defines a single retention cleanup rule.
type RetentionRule struct {
	Name      string
	Retention time.Duration
	Cleanup   func(ctx context.Context, before time.Time) (int64, error)
}

// RetentionEnforcer provides retention rules for enforcement.
type RetentionEnforcer interface {
	Rules() []RetentionRule
}

// DefaultRetentionEnforcer creates a retention enforcer from the service's policy
// and pluggable cleanup functions.
type DefaultRetentionEnforcer struct {
	rules []RetentionRule
}

// NewRetentionEnforcer creates a new retention enforcer with the given rules.
func NewRetentionEnforcer(rules []RetentionRule) *DefaultRetentionEnforcer {
	return &DefaultRetentionEnforcer{rules: rules}
}

// Rules returns the configured retention rules.
func (e *DefaultRetentionEnforcer) Rules() []RetentionRule {
	if e == nil {
		return nil
	}
	return e.rules
}
