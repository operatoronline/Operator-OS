package beta

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// API provides HTTP handlers for beta launch management.
type API struct {
	store    *Store
	checker  *ReadinessChecker
}

// NewAPI creates a new beta API handler.
func NewAPI(store *Store, checker *ReadinessChecker) *API {
	return &API{
		store:   store,
		checker: checker,
	}
}

// RegisterRoutes registers all beta API routes on the given mux.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// Invite management (admin)
	mux.HandleFunc("/api/v1/beta/invites", a.handleInvites)
	mux.HandleFunc("/api/v1/beta/invites/", a.handleInviteByID)

	// Invite redemption (public)
	mux.HandleFunc("/api/v1/beta/redeem", a.handleRedeem)

	// Feature flags (admin)
	mux.HandleFunc("/api/v1/beta/flags", a.handleFlags)
	mux.HandleFunc("/api/v1/beta/flags/", a.handleFlagByName)
	mux.HandleFunc("/api/v1/beta/flags/check", a.handleFlagCheck)

	// Waitlist (mixed)
	mux.HandleFunc("/api/v1/beta/waitlist", a.handleWaitlist)
	mux.HandleFunc("/api/v1/beta/waitlist/", a.handleWaitlistByEmail)

	// Readiness (admin)
	mux.HandleFunc("/api/v1/beta/readiness", a.handleReadiness)
}

// --- Invite Handlers ---

type createInviteRequest struct {
	Email    string `json:"email,omitempty"`
	MaxUses  int    `json:"max_uses,omitempty"`
	Note     string `json:"note,omitempty"`
	TTLHours int    `json:"ttl_hours,omitempty"` // 0 = no expiry
}

type redeemRequest struct {
	Code  string `json:"code"`
	Email string `json:"email"`
}

func (a *API) handleInvites(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		invites, err := a.store.ListInvites(status)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if invites == nil {
			invites = []*InviteCode{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"invites": invites,
			"count":   len(invites),
		})

	case http.MethodPost:
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		var req createInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}

		invite := &InviteCode{
			CreatedBy: userID,
			Email:     req.Email,
			MaxUses:   req.MaxUses,
			Note:      req.Note,
		}
		if req.TTLHours > 0 {
			invite.ExpiresAt = time.Now().UTC().Add(time.Duration(req.TTLHours) * time.Hour)
		}

		if err := a.store.CreateInvite(invite); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, invite)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) handleInviteByID(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	// Extract ID from path: /api/v1/beta/invites/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/beta/invites/"), "/")
	id := parts[0]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invite ID required"})
		return
	}

	// Check for /revoke action
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		invite, err := a.store.GetInvite(id)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrInviteNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, invite)

	case r.Method == http.MethodPost && action == "revoke":
		if err := a.store.RevokeInvite(id); err != nil {
			status := http.StatusInternalServerError
			if err == ErrInviteNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) handleRedeem(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req redeemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	invite, err := a.store.RedeemInvite(req.Code, req.Email)
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case ErrInviteNotFound:
			status = http.StatusNotFound
		case ErrInviteExpired:
			status = http.StatusGone
		case ErrInviteUsed, ErrInviteExhausted:
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "redeemed",
		"invite":  invite,
	})
}

// --- Feature Flag Handlers ---

type createFlagRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Enabled     bool     `json:"enabled"`
	RolloutPct  int      `json:"rollout_pct"`
	Plans       []string `json:"plans,omitempty"`
	UserIDs     []string `json:"user_ids,omitempty"`
}

type checkFlagRequest struct {
	FlagName string `json:"flag_name"`
	UserID   string `json:"user_id"`
	UserPlan string `json:"user_plan"`
}

func (a *API) handleFlags(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		flags, err := a.store.ListFlags()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if flags == nil {
			flags = []*FeatureFlag{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"flags": flags,
			"count": len(flags),
		})

	case http.MethodPost:
		var req createFlagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
			return
		}

		flag := &FeatureFlag{
			Name:        req.Name,
			Description: req.Description,
			Enabled:     req.Enabled,
			RolloutPct:  req.RolloutPct,
			Plans:       req.Plans,
			UserIDs:     req.UserIDs,
		}
		if err := a.store.CreateFlag(flag); err != nil {
			status := http.StatusInternalServerError
			if err == ErrDuplicateFlag {
				status = http.StatusConflict
			}
			if err == ErrInvalidRollout {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, flag)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) handleFlagByName(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/beta/flags/")
	if name == "" || name == "check" {
		// "check" is handled by handleFlagCheck
		if name == "check" {
			a.handleFlagCheck(w, r)
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "flag name required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		flag, err := a.store.GetFlag(name)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrFlagNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, flag)

	case http.MethodPut:
		var req createFlagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}

		existing, err := a.store.GetFlag(name)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrFlagNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		existing.Description = req.Description
		existing.Enabled = req.Enabled
		existing.RolloutPct = req.RolloutPct
		existing.Plans = req.Plans
		existing.UserIDs = req.UserIDs

		if err := a.store.UpdateFlag(existing); err != nil {
			status := http.StatusInternalServerError
			if err == ErrInvalidRollout {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, existing)

	case http.MethodDelete:
		if err := a.store.DeleteFlag(name); err != nil {
			status := http.StatusInternalServerError
			if err == ErrFlagNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) handleFlagCheck(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req checkFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.FlagName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "flag_name required"})
		return
	}
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}

	flag, err := a.store.GetFlag(req.FlagName)
	if err != nil {
		if err == ErrFlagNotFound {
			writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "reason": "flag not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	enabled := flag.IsEnabledForUser(req.UserID, req.UserPlan)
	writeJSON(w, http.StatusOK, map[string]any{
		"flag_name": req.FlagName,
		"user_id":   req.UserID,
		"enabled":   enabled,
	})
}

// --- Waitlist Handlers ---

type waitlistSignupRequest struct {
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
}

func (a *API) handleWaitlist(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		entries, err := a.store.ListWaitlist(status)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if entries == nil {
			entries = []*WaitlistEntry{}
		}

		count, _ := a.store.CountWaitlist("")
		pendingCount, _ := a.store.CountWaitlist(WaitlistStatusPending)

		writeJSON(w, http.StatusOK, map[string]any{
			"entries":       entries,
			"count":         len(entries),
			"total":         count,
			"pending_count": pendingCount,
		})

	case http.MethodPost:
		var req waitlistSignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if req.Email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
			return
		}

		entry := &WaitlistEntry{
			Email:  req.Email,
			Name:   req.Name,
			Source: req.Source,
		}
		if err := a.store.AddToWaitlist(entry); err != nil {
			status := http.StatusInternalServerError
			if err == ErrWaitlistDuplicate {
				status = http.StatusConflict
			}
			if err == ErrInvalidEmail {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, entry)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) handleWaitlistByEmail(w http.ResponseWriter, r *http.Request) {
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "beta store not configured"})
		return
	}

	email := strings.TrimPrefix(r.URL.Path, "/api/v1/beta/waitlist/")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		entry, err := a.store.GetWaitlistEntry(email)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrWaitlistNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, entry)

	case http.MethodDelete:
		if err := a.store.RemoveFromWaitlist(email); err != nil {
			status := http.StatusInternalServerError
			if err == ErrWaitlistNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// --- Readiness Handler ---

func (a *API) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if a.checker == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "readiness checker not configured"})
		return
	}

	report := a.checker.Run()
	status := http.StatusOK
	if !report.Ready {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, report)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
