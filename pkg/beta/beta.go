// Package beta provides beta launch infrastructure for Operator OS.
//
// It includes invite code management, feature flags, rollout controls,
// and launch readiness checks for controlled beta rollouts.
package beta

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Errors returned by beta package functions.
var (
	ErrNilDB              = errors.New("beta: nil database")
	ErrInviteNotFound     = errors.New("beta: invite code not found")
	ErrInviteExpired      = errors.New("beta: invite code expired")
	ErrInviteUsed         = errors.New("beta: invite code already used")
	ErrInviteExhausted    = errors.New("beta: invite code max uses reached")
	ErrInvalidCode        = errors.New("beta: invalid invite code")
	ErrFlagNotFound       = errors.New("beta: feature flag not found")
	ErrDuplicateFlag      = errors.New("beta: feature flag already exists")
	ErrInvalidFlag        = errors.New("beta: invalid feature flag")
	ErrInvalidRollout     = errors.New("beta: invalid rollout percentage")
	ErrWaitlistDuplicate  = errors.New("beta: email already on waitlist")
	ErrWaitlistNotFound   = errors.New("beta: waitlist entry not found")
	ErrNotReady           = errors.New("beta: system not ready for launch")
	ErrInvalidEmail       = errors.New("beta: invalid email")
	ErrInvalidStatus      = errors.New("beta: invalid status")
	ErrEmptyID            = errors.New("beta: empty ID")
)

// InviteCode represents a beta invite code.
type InviteCode struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	CreatedBy string    `json:"created_by"`
	Email     string    `json:"email,omitempty"`     // If targeted to a specific email
	MaxUses   int       `json:"max_uses"`            // 0 = unlimited
	UseCount  int       `json:"use_count"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Note      string    `json:"note,omitempty"`
	Status    string    `json:"status"` // active, revoked, expired
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Invite status constants.
const (
	InviteStatusActive  = "active"
	InviteStatusRevoked = "revoked"
	InviteStatusExpired = "expired"
)

// ValidInviteStatus checks if a status string is valid.
func ValidInviteStatus(s string) bool {
	switch s {
	case InviteStatusActive, InviteStatusRevoked, InviteStatusExpired:
		return true
	}
	return false
}

// IsUsable returns true if the invite can still be redeemed.
func (ic *InviteCode) IsUsable() bool {
	if ic.Status != InviteStatusActive {
		return false
	}
	if !ic.ExpiresAt.IsZero() && time.Now().After(ic.ExpiresAt) {
		return false
	}
	if ic.MaxUses > 0 && ic.UseCount >= ic.MaxUses {
		return false
	}
	return true
}

// FeatureFlag represents a feature flag for controlled rollout.
type FeatureFlag struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	RolloutPct  int       `json:"rollout_pct"` // 0-100, percentage of users
	Plans       []string  `json:"plans,omitempty"` // Plan IDs that get this feature
	UserIDs     []string  `json:"user_ids,omitempty"` // Specific user IDs
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsEnabledForUser checks if a feature flag is enabled for a specific user.
// Logic: flag must be globally enabled, then check:
//   1. If user is in the explicit UserIDs list → enabled
//   2. If user's plan is in the Plans list → enabled
//   3. If rollout percentage covers this user (deterministic hash) → enabled
func (ff *FeatureFlag) IsEnabledForUser(userID string, userPlan string) bool {
	if !ff.Enabled {
		return false
	}

	// Check explicit user list
	for _, uid := range ff.UserIDs {
		if uid == userID {
			return true
		}
	}

	// Check plan-based access
	for _, plan := range ff.Plans {
		if plan == userPlan {
			return true
		}
	}

	// Rollout percentage (deterministic by user ID + flag name)
	if ff.RolloutPct >= 100 {
		return true
	}
	if ff.RolloutPct <= 0 {
		return false
	}

	return hashToPercent(userID, ff.Name) < ff.RolloutPct
}

// hashToPercent returns a deterministic 0-99 value for a user+flag combo.
func hashToPercent(userID, flagName string) int {
	// Simple deterministic hash using FNV-like approach.
	h := uint32(2166136261)
	for _, c := range userID + ":" + flagName {
		h ^= uint32(c)
		h *= 16777619
	}
	return int(h % 100)
}

// WaitlistEntry represents a beta waitlist signup.
type WaitlistEntry struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Source    string    `json:"source,omitempty"` // Where they signed up from
	Status    string    `json:"status"`           // pending, invited, joined, removed
	InviteID  string    `json:"invite_id,omitempty"` // Invite code sent to them
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Waitlist status constants.
const (
	WaitlistStatusPending  = "pending"
	WaitlistStatusInvited  = "invited"
	WaitlistStatusJoined   = "joined"
	WaitlistStatusRemoved  = "removed"
)

// ValidWaitlistStatus checks if a status string is valid.
func ValidWaitlistStatus(s string) bool {
	switch s {
	case WaitlistStatusPending, WaitlistStatusInvited, WaitlistStatusJoined, WaitlistStatusRemoved:
		return true
	}
	return false
}

// ReadinessCheck represents a single system readiness check.
type ReadinessCheck struct {
	Name     string `json:"name"`
	Category string `json:"category"` // database, auth, billing, integrations, security, monitoring
	Status   string `json:"status"`   // pass, fail, warn
	Message  string `json:"message,omitempty"`
	Critical bool   `json:"critical"` // If true, blocks launch
}

// Readiness check status constants.
const (
	CheckPass = "pass"
	CheckFail = "fail"
	CheckWarn = "warn"
)

// ReadinessReport is the result of a full readiness assessment.
type ReadinessReport struct {
	Ready     bool              `json:"ready"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    []ReadinessCheck  `json:"checks"`
	Summary   ReadinessSummary  `json:"summary"`
}

// ReadinessSummary aggregates check results.
type ReadinessSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Critical int `json:"critical_failures"`
}

// ReadinessCheckFunc is a function that performs a readiness check.
type ReadinessCheckFunc func() ReadinessCheck

// Store provides persistence for beta launch data.
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore creates a new beta store, initializing the schema.
func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("beta: init schema: %w", err)
	}
	return s, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS beta_invite_codes (
		id         TEXT PRIMARY KEY,
		code       TEXT NOT NULL UNIQUE,
		created_by TEXT NOT NULL,
		email      TEXT DEFAULT '',
		max_uses   INTEGER DEFAULT 0,
		use_count  INTEGER DEFAULT 0,
		expires_at TEXT DEFAULT '',
		note       TEXT DEFAULT '',
		status     TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_invite_codes_code ON beta_invite_codes(code);
	CREATE INDEX IF NOT EXISTS idx_invite_codes_status ON beta_invite_codes(status);
	CREATE INDEX IF NOT EXISTS idx_invite_codes_email ON beta_invite_codes(email);

	CREATE TABLE IF NOT EXISTS beta_feature_flags (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '',
		enabled     INTEGER DEFAULT 0,
		rollout_pct INTEGER DEFAULT 0,
		plans       TEXT DEFAULT '[]',
		user_ids    TEXT DEFAULT '[]',
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_feature_flags_name ON beta_feature_flags(name);

	CREATE TABLE IF NOT EXISTS beta_waitlist (
		id        TEXT PRIMARY KEY,
		email     TEXT NOT NULL UNIQUE,
		name      TEXT DEFAULT '',
		source    TEXT DEFAULT '',
		status    TEXT NOT NULL DEFAULT 'pending',
		invite_id TEXT DEFAULT '',
		note      TEXT DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_waitlist_email ON beta_waitlist(email);
	CREATE INDEX IF NOT EXISTS idx_waitlist_status ON beta_waitlist(status);
	`
	_, err := s.db.Exec(schema)
	return err
}

// generateID creates a random hex ID.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCode creates a beta invite code (format: BETA-XXXX-XXXX).
func generateCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	code := hex.EncodeToString(b)
	return fmt.Sprintf("BETA-%s-%s", strings.ToUpper(code[:4]), strings.ToUpper(code[4:]))
}

// --- Invite Code Operations ---

// CreateInvite creates a new invite code.
func (s *Store) CreateInvite(invite *InviteCode) error {
	if invite == nil {
		return ErrInvalidCode
	}
	if invite.CreatedBy == "" {
		return fmt.Errorf("%w: created_by required", ErrInvalidCode)
	}

	now := time.Now().UTC()
	if invite.ID == "" {
		invite.ID = generateID()
	}
	if invite.Code == "" {
		invite.Code = generateCode()
	}
	if invite.Status == "" {
		invite.Status = InviteStatusActive
	}
	if !ValidInviteStatus(invite.Status) {
		return ErrInvalidStatus
	}
	invite.CreatedAt = now
	invite.UpdatedAt = now

	expiresStr := ""
	if !invite.ExpiresAt.IsZero() {
		expiresStr = invite.ExpiresAt.Format(time.RFC3339)
	}

	_, err := s.db.Exec(`
		INSERT INTO beta_invite_codes (id, code, created_by, email, max_uses, use_count, expires_at, note, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invite.ID, invite.Code, invite.CreatedBy, invite.Email,
		invite.MaxUses, invite.UseCount, expiresStr, invite.Note,
		invite.Status, invite.CreatedAt.Format(time.RFC3339), invite.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetInvite retrieves an invite by ID.
func (s *Store) GetInvite(id string) (*InviteCode, error) {
	if id == "" {
		return nil, ErrEmptyID
	}
	return s.scanInvite(s.db.QueryRow(`SELECT id, code, created_by, email, max_uses, use_count, expires_at, note, status, created_at, updated_at FROM beta_invite_codes WHERE id = ?`, id))
}

// GetInviteByCode retrieves an invite by code string.
func (s *Store) GetInviteByCode(code string) (*InviteCode, error) {
	if code == "" {
		return nil, ErrInvalidCode
	}
	return s.scanInvite(s.db.QueryRow(`SELECT id, code, created_by, email, max_uses, use_count, expires_at, note, status, created_at, updated_at FROM beta_invite_codes WHERE code = ?`, code))
}

// ListInvites returns all invite codes, optionally filtered by status.
func (s *Store) ListInvites(status string) ([]*InviteCode, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		if !ValidInviteStatus(status) {
			return nil, ErrInvalidStatus
		}
		rows, err = s.db.Query(`SELECT id, code, created_by, email, max_uses, use_count, expires_at, note, status, created_at, updated_at FROM beta_invite_codes WHERE status = ? ORDER BY created_at DESC`, status)
	} else {
		rows, err = s.db.Query(`SELECT id, code, created_by, email, max_uses, use_count, expires_at, note, status, created_at, updated_at FROM beta_invite_codes ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []*InviteCode
	for rows.Next() {
		ic, err := s.scanInviteRow(rows)
		if err != nil {
			return nil, err
		}
		invites = append(invites, ic)
	}
	return invites, rows.Err()
}

// RedeemInvite validates and redeems an invite code, incrementing its use count.
func (s *Store) RedeemInvite(code string, email string) (*InviteCode, error) {
	if code == "" {
		return nil, ErrInvalidCode
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ic, err := s.GetInviteByCode(code)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	if ic.Status == InviteStatusRevoked {
		return nil, ErrInviteUsed
	}

	if !ic.ExpiresAt.IsZero() && time.Now().After(ic.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	if ic.MaxUses > 0 && ic.UseCount >= ic.MaxUses {
		return nil, ErrInviteExhausted
	}

	// If targeted to a specific email, validate
	if ic.Email != "" && !strings.EqualFold(ic.Email, email) {
		return nil, ErrInviteNotFound
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE beta_invite_codes SET use_count = use_count + 1, updated_at = ? WHERE id = ?`,
		now.Format(time.RFC3339), ic.ID)
	if err != nil {
		return nil, err
	}

	ic.UseCount++
	ic.UpdatedAt = now
	return ic, nil
}

// RevokeInvite marks an invite as revoked.
func (s *Store) RevokeInvite(id string) error {
	if id == "" {
		return ErrEmptyID
	}
	res, err := s.db.Exec(`UPDATE beta_invite_codes SET status = ?, updated_at = ? WHERE id = ?`,
		InviteStatusRevoked, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInviteNotFound
	}
	return nil
}

// CountInvites returns the number of invites, optionally filtered by status.
func (s *Store) CountInvites(status string) (int64, error) {
	var count int64
	if status != "" {
		err := s.db.QueryRow(`SELECT COUNT(*) FROM beta_invite_codes WHERE status = ?`, status).Scan(&count)
		return count, err
	}
	err := s.db.QueryRow(`SELECT COUNT(*) FROM beta_invite_codes`).Scan(&count)
	return count, err
}

func (s *Store) scanInvite(row *sql.Row) (*InviteCode, error) {
	ic := &InviteCode{}
	var expiresStr, createdStr, updatedStr string
	err := row.Scan(&ic.ID, &ic.Code, &ic.CreatedBy, &ic.Email, &ic.MaxUses, &ic.UseCount,
		&expiresStr, &ic.Note, &ic.Status, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}
	ic.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	ic.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	if expiresStr != "" {
		ic.ExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}
	return ic, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) scanInviteRow(row rowScanner) (*InviteCode, error) {
	ic := &InviteCode{}
	var expiresStr, createdStr, updatedStr string
	err := row.Scan(&ic.ID, &ic.Code, &ic.CreatedBy, &ic.Email, &ic.MaxUses, &ic.UseCount,
		&expiresStr, &ic.Note, &ic.Status, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	ic.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	ic.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	if expiresStr != "" {
		ic.ExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}
	return ic, nil
}

// --- Feature Flag Operations ---

// CreateFlag creates a new feature flag.
func (s *Store) CreateFlag(flag *FeatureFlag) error {
	if flag == nil {
		return ErrInvalidFlag
	}
	if flag.Name == "" {
		return fmt.Errorf("%w: name required", ErrInvalidFlag)
	}
	if flag.RolloutPct < 0 || flag.RolloutPct > 100 {
		return ErrInvalidRollout
	}

	now := time.Now().UTC()
	if flag.ID == "" {
		flag.ID = generateID()
	}
	flag.CreatedAt = now
	flag.UpdatedAt = now

	plansJSON := "[]"
	if len(flag.Plans) > 0 {
		plansJSON = marshalStringSlice(flag.Plans)
	}
	userIDsJSON := "[]"
	if len(flag.UserIDs) > 0 {
		userIDsJSON = marshalStringSlice(flag.UserIDs)
	}

	_, err := s.db.Exec(`
		INSERT INTO beta_feature_flags (id, name, description, enabled, rollout_pct, plans, user_ids, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		flag.ID, flag.Name, flag.Description, boolToInt(flag.Enabled),
		flag.RolloutPct, plansJSON, userIDsJSON,
		flag.CreatedAt.Format(time.RFC3339), flag.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return ErrDuplicateFlag
	}
	return err
}

// GetFlag retrieves a feature flag by name.
func (s *Store) GetFlag(name string) (*FeatureFlag, error) {
	if name == "" {
		return nil, ErrInvalidFlag
	}
	return s.scanFlag(s.db.QueryRow(`SELECT id, name, description, enabled, rollout_pct, plans, user_ids, created_at, updated_at FROM beta_feature_flags WHERE name = ?`, name))
}

// GetFlagByID retrieves a feature flag by ID.
func (s *Store) GetFlagByID(id string) (*FeatureFlag, error) {
	if id == "" {
		return nil, ErrEmptyID
	}
	return s.scanFlag(s.db.QueryRow(`SELECT id, name, description, enabled, rollout_pct, plans, user_ids, created_at, updated_at FROM beta_feature_flags WHERE id = ?`, id))
}

// ListFlags returns all feature flags.
func (s *Store) ListFlags() ([]*FeatureFlag, error) {
	rows, err := s.db.Query(`SELECT id, name, description, enabled, rollout_pct, plans, user_ids, created_at, updated_at FROM beta_feature_flags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []*FeatureFlag
	for rows.Next() {
		ff, err := s.scanFlagRow(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, ff)
	}
	return flags, rows.Err()
}

// UpdateFlag updates a feature flag.
func (s *Store) UpdateFlag(flag *FeatureFlag) error {
	if flag == nil {
		return ErrInvalidFlag
	}
	if flag.ID == "" {
		return ErrEmptyID
	}
	if flag.RolloutPct < 0 || flag.RolloutPct > 100 {
		return ErrInvalidRollout
	}

	now := time.Now().UTC()
	flag.UpdatedAt = now

	plansJSON := marshalStringSlice(flag.Plans)
	userIDsJSON := marshalStringSlice(flag.UserIDs)

	res, err := s.db.Exec(`
		UPDATE beta_feature_flags SET name = ?, description = ?, enabled = ?, rollout_pct = ?, plans = ?, user_ids = ?, updated_at = ?
		WHERE id = ?`,
		flag.Name, flag.Description, boolToInt(flag.Enabled),
		flag.RolloutPct, plansJSON, userIDsJSON,
		flag.UpdatedAt.Format(time.RFC3339), flag.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrFlagNotFound
	}
	return nil
}

// DeleteFlag deletes a feature flag by name.
func (s *Store) DeleteFlag(name string) error {
	if name == "" {
		return ErrInvalidFlag
	}
	res, err := s.db.Exec(`DELETE FROM beta_feature_flags WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrFlagNotFound
	}
	return nil
}

func (s *Store) scanFlag(row *sql.Row) (*FeatureFlag, error) {
	ff := &FeatureFlag{}
	var enabled int
	var plansJSON, userIDsJSON, createdStr, updatedStr string
	err := row.Scan(&ff.ID, &ff.Name, &ff.Description, &enabled, &ff.RolloutPct,
		&plansJSON, &userIDsJSON, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFlagNotFound
		}
		return nil, err
	}
	ff.Enabled = enabled != 0
	ff.Plans = unmarshalStringSlice(plansJSON)
	ff.UserIDs = unmarshalStringSlice(userIDsJSON)
	ff.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	ff.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return ff, nil
}

func (s *Store) scanFlagRow(row rowScanner) (*FeatureFlag, error) {
	ff := &FeatureFlag{}
	var enabled int
	var plansJSON, userIDsJSON, createdStr, updatedStr string
	err := row.Scan(&ff.ID, &ff.Name, &ff.Description, &enabled, &ff.RolloutPct,
		&plansJSON, &userIDsJSON, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	ff.Enabled = enabled != 0
	ff.Plans = unmarshalStringSlice(plansJSON)
	ff.UserIDs = unmarshalStringSlice(userIDsJSON)
	ff.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	ff.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return ff, nil
}

// --- Waitlist Operations ---

// AddToWaitlist adds an email to the beta waitlist.
func (s *Store) AddToWaitlist(entry *WaitlistEntry) error {
	if entry == nil {
		return ErrInvalidEmail
	}
	if entry.Email == "" {
		return ErrInvalidEmail
	}
	if !strings.Contains(entry.Email, "@") {
		return ErrInvalidEmail
	}

	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.Status == "" {
		entry.Status = WaitlistStatusPending
	}
	if !ValidWaitlistStatus(entry.Status) {
		return ErrInvalidStatus
	}
	entry.Email = strings.ToLower(strings.TrimSpace(entry.Email))
	entry.CreatedAt = now
	entry.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO beta_waitlist (id, email, name, source, status, invite_id, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Email, entry.Name, entry.Source,
		entry.Status, entry.InviteID, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return ErrWaitlistDuplicate
	}
	return err
}

// GetWaitlistEntry retrieves a waitlist entry by email.
func (s *Store) GetWaitlistEntry(email string) (*WaitlistEntry, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}
	email = strings.ToLower(strings.TrimSpace(email))
	return s.scanWaitlist(s.db.QueryRow(`SELECT id, email, name, source, status, invite_id, note, created_at, updated_at FROM beta_waitlist WHERE email = ?`, email))
}

// ListWaitlist returns waitlist entries, optionally filtered by status.
func (s *Store) ListWaitlist(status string) ([]*WaitlistEntry, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		if !ValidWaitlistStatus(status) {
			return nil, ErrInvalidStatus
		}
		rows, err = s.db.Query(`SELECT id, email, name, source, status, invite_id, note, created_at, updated_at FROM beta_waitlist WHERE status = ? ORDER BY created_at ASC`, status)
	} else {
		rows, err = s.db.Query(`SELECT id, email, name, source, status, invite_id, note, created_at, updated_at FROM beta_waitlist ORDER BY created_at ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*WaitlistEntry
	for rows.Next() {
		we, err := s.scanWaitlistRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, we)
	}
	return entries, rows.Err()
}

// UpdateWaitlistStatus updates the status and optionally the invite ID of a waitlist entry.
func (s *Store) UpdateWaitlistStatus(email string, status string, inviteID string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if !ValidWaitlistStatus(status) {
		return ErrInvalidStatus
	}
	email = strings.ToLower(strings.TrimSpace(email))
	now := time.Now().UTC()

	res, err := s.db.Exec(`UPDATE beta_waitlist SET status = ?, invite_id = ?, updated_at = ? WHERE email = ?`,
		status, inviteID, now.Format(time.RFC3339), email)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrWaitlistNotFound
	}
	return nil
}

// RemoveFromWaitlist removes an entry from the waitlist.
func (s *Store) RemoveFromWaitlist(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	email = strings.ToLower(strings.TrimSpace(email))
	res, err := s.db.Exec(`DELETE FROM beta_waitlist WHERE email = ?`, email)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrWaitlistNotFound
	}
	return nil
}

// CountWaitlist returns the number of waitlist entries, optionally filtered by status.
func (s *Store) CountWaitlist(status string) (int64, error) {
	var count int64
	if status != "" {
		err := s.db.QueryRow(`SELECT COUNT(*) FROM beta_waitlist WHERE status = ?`, status).Scan(&count)
		return count, err
	}
	err := s.db.QueryRow(`SELECT COUNT(*) FROM beta_waitlist`).Scan(&count)
	return count, err
}

func (s *Store) scanWaitlist(row *sql.Row) (*WaitlistEntry, error) {
	we := &WaitlistEntry{}
	var createdStr, updatedStr string
	err := row.Scan(&we.ID, &we.Email, &we.Name, &we.Source, &we.Status, &we.InviteID, &we.Note, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWaitlistNotFound
		}
		return nil, err
	}
	we.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	we.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return we, nil
}

func (s *Store) scanWaitlistRow(row rowScanner) (*WaitlistEntry, error) {
	we := &WaitlistEntry{}
	var createdStr, updatedStr string
	err := row.Scan(&we.ID, &we.Email, &we.Name, &we.Source, &we.Status, &we.InviteID, &we.Note, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	we.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	we.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return we, nil
}

// Close is a no-op (caller owns the database connection).
func (s *Store) Close() error {
	return nil
}

// --- Helpers ---

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func marshalStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	// Simple JSON marshal without import
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func unmarshalStringSlice(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	// Simple JSON unmarshal without import
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
