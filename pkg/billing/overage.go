package billing

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// OverageLevel describes how far over a plan limit a user is.
type OverageLevel string

const (
	// OverageLevelNone means usage is within normal bounds.
	OverageLevelNone OverageLevel = "none"
	// OverageLevelWarning means usage is approaching the limit (soft cap).
	OverageLevelWarning OverageLevel = "warning"
	// OverageLevelSoftCap means usage has crossed the warning threshold but is below the hard cap.
	OverageLevelSoftCap OverageLevel = "soft_cap"
	// OverageLevelHardCap means usage has exceeded the limit; service is throttled.
	OverageLevelHardCap OverageLevel = "hard_cap"
	// OverageLevelBlocked means usage is far beyond the limit; requests are blocked.
	OverageLevelBlocked OverageLevel = "blocked"
)

// OverageAction describes what enforcement action to take.
type OverageAction string

const (
	ActionNone           OverageAction = "none"
	ActionWarn           OverageAction = "warn"
	ActionDowngradeModel OverageAction = "downgrade_model"
	ActionThrottle       OverageAction = "throttle"
	ActionBlock          OverageAction = "block"
)

// OverageThresholds defines the percentage thresholds for each overage level.
// All values are fractions (e.g., 0.80 = 80%).
type OverageThresholds struct {
	// Warning is the percentage of limit at which warnings begin (default 0.80).
	Warning float64 `json:"warning"`
	// SoftCap is the percentage at which soft enforcement begins (default 0.90).
	SoftCap float64 `json:"soft_cap"`
	// HardCap is the percentage at which throttling begins (default 1.00).
	HardCap float64 `json:"hard_cap"`
	// BlockAt is the percentage at which requests are blocked (default 1.20).
	BlockAt float64 `json:"block_at"`
}

// DefaultOverageThresholds returns the default threshold configuration.
func DefaultOverageThresholds() OverageThresholds {
	return OverageThresholds{
		Warning: 0.80,
		SoftCap: 0.90,
		HardCap: 1.00,
		BlockAt: 1.20,
	}
}

// Validate checks that thresholds are ordered correctly.
func (t OverageThresholds) Validate() error {
	if t.Warning <= 0 || t.Warning >= 1 {
		return fmt.Errorf("billing: warning threshold must be between 0 and 1, got %f", t.Warning)
	}
	if t.SoftCap <= t.Warning {
		return fmt.Errorf("billing: soft_cap (%f) must be greater than warning (%f)", t.SoftCap, t.Warning)
	}
	if t.HardCap <= t.SoftCap {
		return fmt.Errorf("billing: hard_cap (%f) must be greater than soft_cap (%f)", t.HardCap, t.SoftCap)
	}
	if t.BlockAt <= t.HardCap {
		return fmt.Errorf("billing: block_at (%f) must be greater than hard_cap (%f)", t.BlockAt, t.HardCap)
	}
	return nil
}

// OverageStatus represents the current overage state for a resource.
type OverageStatus struct {
	Resource    string        `json:"resource"`
	Usage       int64         `json:"usage"`
	Limit       int64         `json:"limit"`
	Percentage  float64       `json:"percentage"`
	Level       OverageLevel  `json:"level"`
	Action      OverageAction `json:"action"`
	Message     string        `json:"message,omitempty"`
	FallbackModel string      `json:"fallback_model,omitempty"`
}

// ThrottleConfig defines throttling behaviour when the hard cap is exceeded.
type ThrottleConfig struct {
	// FallbackModel is the model to downgrade to when throttled.
	FallbackModel string `json:"fallback_model"`
	// DelayMs is the artificial delay added to requests when throttled (milliseconds).
	DelayMs int `json:"delay_ms"`
	// ReducedRatePercent is the percentage of normal rate limit to apply when throttled.
	// E.g., 50 means rate limit is halved.
	ReducedRatePercent int `json:"reduced_rate_percent"`
}

// DefaultThrottleConfig returns the default throttle configuration.
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		FallbackModel:      "gpt-4o-mini",
		DelayMs:            500,
		ReducedRatePercent: 50,
	}
}

// OverageEnforcer checks user usage against plan limits and determines enforcement actions.
type OverageEnforcer struct {
	mu         sync.RWMutex
	usageStore UsageStore
	subStore   SubscriptionStore
	catalogue  *Catalogue
	thresholds OverageThresholds
	throttle   ThrottleConfig
}

// OverageEnforcerConfig holds configuration for the OverageEnforcer.
type OverageEnforcerConfig struct {
	UsageStore UsageStore
	SubStore   SubscriptionStore
	Catalogue  *Catalogue
	Thresholds *OverageThresholds
	Throttle   *ThrottleConfig
}

// NewOverageEnforcer creates an OverageEnforcer with the given configuration.
func NewOverageEnforcer(cfg OverageEnforcerConfig) (*OverageEnforcer, error) {
	if cfg.UsageStore == nil {
		return nil, fmt.Errorf("billing: usage store is required for overage enforcer")
	}
	if cfg.Catalogue == nil {
		return nil, fmt.Errorf("billing: catalogue is required for overage enforcer")
	}

	thresholds := DefaultOverageThresholds()
	if cfg.Thresholds != nil {
		thresholds = *cfg.Thresholds
	}
	if err := thresholds.Validate(); err != nil {
		return nil, err
	}

	throttle := DefaultThrottleConfig()
	if cfg.Throttle != nil {
		throttle = *cfg.Throttle
	}

	return &OverageEnforcer{
		usageStore: cfg.UsageStore,
		subStore:   cfg.SubStore,
		catalogue:  cfg.Catalogue,
		thresholds: thresholds,
		throttle:   throttle,
	}, nil
}

// CheckTokens checks the user's token usage against their plan limit.
func (e *OverageEnforcer) CheckTokens(userID string, periodStart time.Time, plan *Plan) (*OverageStatus, error) {
	if plan == nil {
		return nil, fmt.Errorf("billing: plan is nil")
	}

	limit := plan.Limits.MaxTokensPerMonth
	if IsUnlimited(limit) {
		return &OverageStatus{
			Resource:   "tokens",
			Usage:      0,
			Limit:      0,
			Percentage: 0,
			Level:      OverageLevelNone,
			Action:     ActionNone,
		}, nil
	}

	usage, err := e.usageStore.GetCurrentPeriodUsage(userID, periodStart)
	if err != nil {
		return nil, fmt.Errorf("billing: check token usage: %w", err)
	}

	return e.evaluate("tokens", usage, limit), nil
}

// CheckMessages checks the user's message count against their plan limit.
func (e *OverageEnforcer) CheckMessages(userID string, periodStart time.Time, plan *Plan) (*OverageStatus, error) {
	if plan == nil {
		return nil, fmt.Errorf("billing: plan is nil")
	}

	limit := plan.Limits.MaxMessagesPerMonth
	if IsUnlimited(limit) {
		return &OverageStatus{
			Resource:   "messages",
			Usage:      0,
			Limit:      0,
			Percentage: 0,
			Level:      OverageLevelNone,
			Action:     ActionNone,
		}, nil
	}

	usage, err := e.usageStore.GetCurrentPeriodMessages(userID, periodStart)
	if err != nil {
		return nil, fmt.Errorf("billing: check message usage: %w", err)
	}

	return e.evaluate("messages", usage, limit), nil
}

// CheckAll checks both token and message usage, returning the more severe status.
func (e *OverageEnforcer) CheckAll(userID string, periodStart time.Time, plan *Plan) (*OverageStatus, error) {
	tokenStatus, err := e.CheckTokens(userID, periodStart, plan)
	if err != nil {
		return nil, err
	}

	messageStatus, err := e.CheckMessages(userID, periodStart, plan)
	if err != nil {
		return nil, err
	}

	// Return the more severe status.
	if levelSeverity(messageStatus.Level) > levelSeverity(tokenStatus.Level) {
		return messageStatus, nil
	}
	return tokenStatus, nil
}

// CheckUser resolves the user's plan from their subscription and checks all limits.
func (e *OverageEnforcer) CheckUser(userID string) (*OverageStatus, error) {
	plan, periodStart := e.resolvePlan(userID)
	return e.CheckAll(userID, periodStart, plan)
}

// GetFullStatus returns overage status for all resources.
func (e *OverageEnforcer) GetFullStatus(userID string) ([]*OverageStatus, error) {
	plan, periodStart := e.resolvePlan(userID)

	tokenStatus, err := e.CheckTokens(userID, periodStart, plan)
	if err != nil {
		return nil, err
	}

	messageStatus, err := e.CheckMessages(userID, periodStart, plan)
	if err != nil {
		return nil, err
	}

	return []*OverageStatus{tokenStatus, messageStatus}, nil
}

// evaluate computes the overage level and action for a given resource.
func (e *OverageEnforcer) evaluate(resource string, usage, limit int64) *OverageStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pct := float64(usage) / float64(limit)

	status := &OverageStatus{
		Resource:   resource,
		Usage:      usage,
		Limit:      limit,
		Percentage: pct,
	}

	switch {
	case pct >= e.thresholds.BlockAt:
		status.Level = OverageLevelBlocked
		status.Action = ActionBlock
		status.Message = fmt.Sprintf("%s usage is %.0f%% of limit — requests blocked until next billing period", resource, pct*100)

	case pct >= e.thresholds.HardCap:
		status.Level = OverageLevelHardCap
		status.Action = ActionThrottle
		status.FallbackModel = e.throttle.FallbackModel
		status.Message = fmt.Sprintf("%s usage is %.0f%% of limit — service throttled, using fallback model", resource, pct*100)

	case pct >= e.thresholds.SoftCap:
		status.Level = OverageLevelSoftCap
		status.Action = ActionDowngradeModel
		status.FallbackModel = e.throttle.FallbackModel
		status.Message = fmt.Sprintf("%s usage is %.0f%% of limit — approaching hard cap, model may be downgraded", resource, pct*100)

	case pct >= e.thresholds.Warning:
		status.Level = OverageLevelWarning
		status.Action = ActionWarn
		status.Message = fmt.Sprintf("%s usage is %.0f%% of limit — consider upgrading your plan", resource, pct*100)

	default:
		status.Level = OverageLevelNone
		status.Action = ActionNone
	}

	return status
}

// resolvePlan determines the user's plan and billing period start.
func (e *OverageEnforcer) resolvePlan(userID string) (*Plan, time.Time) {
	plan := e.catalogue.Get(PlanFree)
	periodStart := beginningOfMonth(time.Now().UTC())

	if e.subStore != nil {
		sub, err := e.subStore.GetByUserID(userID)
		if err == nil && sub != nil && sub.IsActive() {
			if p := e.catalogue.Get(sub.PlanID); p != nil {
				plan = p
			}
			if !sub.CurrentPeriodStart.IsZero() {
				periodStart = sub.CurrentPeriodStart
			}
		}
	}

	return plan, periodStart
}

// levelSeverity returns a numeric severity for comparison (higher = more severe).
func levelSeverity(l OverageLevel) int {
	switch l {
	case OverageLevelNone:
		return 0
	case OverageLevelWarning:
		return 1
	case OverageLevelSoftCap:
		return 2
	case OverageLevelHardCap:
		return 3
	case OverageLevelBlocked:
		return 4
	default:
		return 0
	}
}

// ---------- HTTP Middleware ----------

// OverageMiddleware checks usage limits before allowing requests through.
// It adds X-Overage-Level and X-Overage-Action headers to all responses.
// At the hard cap, it adds an artificial delay. At the block level, it returns 429.
func OverageMiddleware(enforcer *OverageEnforcer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := userIDFromContext(r)
			if userID == "" {
				// No auth context — let the request through (auth middleware handles rejection).
				next.ServeHTTP(w, r)
				return
			}

			status, err := enforcer.CheckUser(userID)
			if err != nil {
				// On error, let the request through (fail open for availability).
				next.ServeHTTP(w, r)
				return
			}

			// Always set informational headers.
			w.Header().Set("X-Overage-Level", string(status.Level))
			w.Header().Set("X-Overage-Action", string(status.Action))
			if status.Message != "" {
				w.Header().Set("X-Overage-Message", status.Message)
			}
			if status.FallbackModel != "" {
				w.Header().Set("X-Overage-Fallback-Model", status.FallbackModel)
			}

			switch status.Action {
			case ActionBlock:
				writeJSON(w, http.StatusTooManyRequests, map[string]any{
					"error":   "usage limit exceeded",
					"code":    "overage_blocked",
					"message": status.Message,
					"overage": status,
				})
				return

			case ActionThrottle:
				// Add artificial delay but allow the request.
				if enforcer.throttle.DelayMs > 0 {
					time.Sleep(time.Duration(enforcer.throttle.DelayMs) * time.Millisecond)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ---------- Overage API ----------

// OverageAPI provides HTTP handlers for overage status checking.
type OverageAPI struct {
	enforcer *OverageEnforcer
}

// NewOverageAPI creates an OverageAPI.
func NewOverageAPI(enforcer *OverageEnforcer) *OverageAPI {
	return &OverageAPI{enforcer: enforcer}
}

// RegisterRoutes registers overage API routes on the given mux.
func (a *OverageAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/billing/overage", a.handleGetOverageStatus)
}

// handleGetOverageStatus returns the user's current overage status for all resources.
func (a *OverageAPI) handleGetOverageStatus(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	statuses, err := a.enforcer.GetFullStatus(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to check overage status",
			"code":  "internal_error",
		})
		return
	}

	// Determine overall level (most severe).
	overallLevel := OverageLevelNone
	overallAction := ActionNone
	for _, s := range statuses {
		if levelSeverity(s.Level) > levelSeverity(overallLevel) {
			overallLevel = s.Level
			overallAction = s.Action
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overall_level":  overallLevel,
		"overall_action": overallAction,
		"resources":      statuses,
	})
}
