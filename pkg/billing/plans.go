// Package billing provides plan definitions, subscription management, and
// usage tracking for Operator OS.
//
// Plans are config-driven: the default tier catalogue ships with the binary,
// but operators can override limits via configuration. The package exposes a
// PlanCatalogue that is safe for concurrent reads and can be hot-reloaded.
package billing

import (
	"encoding/json"
	"fmt"
	"sync"
)

// PlanID identifies a billing plan.
type PlanID string

const (
	PlanFree       PlanID = "free"
	PlanStarter    PlanID = "starter"
	PlanPro        PlanID = "pro"
	PlanEnterprise PlanID = "enterprise"
)

// AllPlanIDs returns the canonical ordering of plan tiers (lowest → highest).
func AllPlanIDs() []PlanID {
	return []PlanID{PlanFree, PlanStarter, PlanPro, PlanEnterprise}
}

// ValidPlanID reports whether id is a recognised plan tier.
func ValidPlanID(id PlanID) bool {
	switch id {
	case PlanFree, PlanStarter, PlanPro, PlanEnterprise:
		return true
	}
	return false
}

// BillingInterval describes how often a plan is billed.
type BillingInterval string

const (
	IntervalMonthly BillingInterval = "monthly"
	IntervalYearly  BillingInterval = "yearly"
	IntervalNone    BillingInterval = "none" // free plans
)

// Plan describes a billing plan's limits and metadata.
type Plan struct {
	// ID is the machine-readable plan identifier.
	ID PlanID `json:"id"`
	// Name is the human-readable plan name.
	Name string `json:"name"`
	// Description briefly describes the plan.
	Description string `json:"description,omitempty"`

	// --- Pricing ---

	// PriceMonthly is the monthly price in cents (USD). 0 for free plans.
	PriceMonthly int64 `json:"price_monthly"`
	// PriceYearly is the yearly price in cents (USD). 0 for free/no-yearly.
	PriceYearly int64 `json:"price_yearly"`

	// --- Feature Limits ---

	Limits PlanLimits `json:"limits"`

	// --- Metadata ---

	// Active indicates whether the plan is available for new sign-ups.
	Active bool `json:"active"`
	// SortOrder controls display ordering (lower = first).
	SortOrder int `json:"sort_order"`
}

// PlanLimits defines quantitative limits for a plan tier.
// A value of 0 means unlimited for that dimension (except where noted).
type PlanLimits struct {
	// MaxAgents is the maximum number of agents a user can create.
	// 0 = unlimited.
	MaxAgents int `json:"max_agents"`
	// MaxMessagesPerMonth is the maximum messages (LLM requests) per calendar month.
	// 0 = unlimited.
	MaxMessagesPerMonth int64 `json:"max_messages_per_month"`
	// MaxTokensPerMonth is the maximum total LLM tokens (input+output) per month.
	// 0 = unlimited.
	MaxTokensPerMonth int64 `json:"max_tokens_per_month"`
	// MaxIntegrations is the maximum connected third-party integrations.
	// 0 = unlimited.
	MaxIntegrations int `json:"max_integrations"`
	// MaxStorageBytes is the maximum media/file storage in bytes.
	// 0 = unlimited.
	MaxStorageBytes int64 `json:"max_storage_bytes"`
	// MaxTeamMembers is the maximum number of team members.
	// 0 = unlimited.
	MaxTeamMembers int `json:"max_team_members"`
	// AllowedModels lists model identifiers the plan may use.
	// Empty slice = all models allowed.
	AllowedModels []string `json:"allowed_models,omitempty"`
	// CustomSkills indicates whether users can create custom skills.
	CustomSkills bool `json:"custom_skills"`
	// APIAccess describes the level of API access: "none", "basic", "full".
	APIAccess string `json:"api_access"`
	// RateRequestsPerMinute is the per-minute rate limit for this plan.
	RateRequestsPerMinute float64 `json:"rate_requests_per_minute"`
	// RateBurstSize is the maximum burst size for rate limiting.
	RateBurstSize int `json:"rate_burst_size"`
	// RateDailyLimit is the daily request cap (0 = unlimited).
	RateDailyLimit int64 `json:"rate_daily_limit"`
}

// DefaultPlans returns the built-in plan catalogue.
func DefaultPlans() map[PlanID]*Plan {
	return map[PlanID]*Plan{
		PlanFree: {
			ID:           PlanFree,
			Name:         "Free",
			Description:  "Get started with the basics",
			PriceMonthly: 0,
			PriceYearly:  0,
			Limits: PlanLimits{
				MaxAgents:             1,
				MaxMessagesPerMonth:   500,
				MaxTokensPerMonth:     500_000,
				MaxIntegrations:       1,
				MaxStorageBytes:       100 * 1024 * 1024, // 100 MB
				MaxTeamMembers:        1,
				AllowedModels:         []string{"gpt-4o-mini"},
				CustomSkills:          false,
				APIAccess:             "none",
				RateRequestsPerMinute: 10,
				RateBurstSize:         15,
				RateDailyLimit:        500,
			},
			Active:    true,
			SortOrder: 0,
		},
		PlanStarter: {
			ID:           PlanStarter,
			Name:         "Starter",
			Description:  "For individuals who need more power",
			PriceMonthly: 900,   // $9/mo
			PriceYearly:  8_640, // $86.40/yr ($7.20/mo)
			Limits: PlanLimits{
				MaxAgents:             3,
				MaxMessagesPerMonth:   5_000,
				MaxTokensPerMonth:     5_000_000,
				MaxIntegrations:       5,
				MaxStorageBytes:       1024 * 1024 * 1024, // 1 GB
				MaxTeamMembers:        1,
				AllowedModels:         []string{"gpt-4o", "gpt-4o-mini", "claude-haiku"},
				CustomSkills:          false,
				APIAccess:             "basic",
				RateRequestsPerMinute: 30,
				RateBurstSize:         50,
				RateDailyLimit:        5_000,
			},
			Active:    true,
			SortOrder: 1,
		},
		PlanPro: {
			ID:           PlanPro,
			Name:         "Pro",
			Description:  "For professionals and small teams",
			PriceMonthly: 2_900,  // $29/mo
			PriceYearly:  27_840, // $278.40/yr ($23.20/mo)
			Limits: PlanLimits{
				MaxAgents:             10,
				MaxMessagesPerMonth:   50_000,
				MaxTokensPerMonth:     50_000_000,
				MaxIntegrations:       20,
				MaxStorageBytes:       10 * 1024 * 1024 * 1024, // 10 GB
				MaxTeamMembers:        5,
				AllowedModels:         nil, // all models
				CustomSkills:          true,
				APIAccess:             "full",
				RateRequestsPerMinute: 60,
				RateBurstSize:         100,
				RateDailyLimit:        50_000,
			},
			Active:    true,
			SortOrder: 2,
		},
		PlanEnterprise: {
			ID:           PlanEnterprise,
			Name:         "Enterprise",
			Description:  "Custom pricing for large organisations",
			PriceMonthly: 0, // custom pricing
			PriceYearly:  0,
			Limits: PlanLimits{
				MaxAgents:             0, // unlimited
				MaxMessagesPerMonth:   0,
				MaxTokensPerMonth:     0,
				MaxIntegrations:       0,
				MaxStorageBytes:       0,
				MaxTeamMembers:        0,
				AllowedModels:         nil,
				CustomSkills:          true,
				APIAccess:             "full",
				RateRequestsPerMinute: 120,
				RateBurstSize:         200,
				RateDailyLimit:        0,
			},
			Active:    true,
			SortOrder: 3,
		},
	}
}

// ---------- Catalogue ----------

// Catalogue holds the active set of plan definitions.
// It is safe for concurrent reads; writes (Replace) serialise via a mutex.
type Catalogue struct {
	mu    sync.RWMutex
	plans map[PlanID]*Plan
}

// NewCatalogue creates a Catalogue pre-loaded with the given plans.
// If plans is nil, DefaultPlans() is used.
func NewCatalogue(plans map[PlanID]*Plan) *Catalogue {
	if plans == nil {
		plans = DefaultPlans()
	}
	return &Catalogue{plans: plans}
}

// Get returns the plan for the given ID, or nil if not found.
func (c *Catalogue) Get(id PlanID) *Plan {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p := c.plans[id]
	if p == nil {
		return nil
	}
	// Return a shallow copy to prevent mutation.
	cp := *p
	return &cp
}

// List returns all plans sorted by SortOrder.
func (c *Catalogue) List() []*Plan {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Plan, 0, len(c.plans))
	for _, p := range c.plans {
		cp := *p
		out = append(out, &cp)
	}
	// Sort by SortOrder, then by Name.
	sortPlans(out)
	return out
}

// ListActive returns only plans marked Active, sorted by SortOrder.
func (c *Catalogue) ListActive() []*Plan {
	all := c.List()
	active := make([]*Plan, 0, len(all))
	for _, p := range all {
		if p.Active {
			active = append(active, p)
		}
	}
	return active
}

// Replace atomically swaps the entire plan set.
func (c *Catalogue) Replace(plans map[PlanID]*Plan) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.plans = plans
}

// Merge adds or replaces individual plans without affecting others.
func (c *Catalogue) Merge(plans map[PlanID]*Plan) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, p := range plans {
		c.plans[id] = p
	}
}

// sortPlans sorts a slice of plans by SortOrder then Name.
func sortPlans(plans []*Plan) {
	for i := 1; i < len(plans); i++ {
		for j := i; j > 0; j-- {
			if plans[j].SortOrder < plans[j-1].SortOrder ||
				(plans[j].SortOrder == plans[j-1].SortOrder && plans[j].Name < plans[j-1].Name) {
				plans[j], plans[j-1] = plans[j-1], plans[j]
			} else {
				break
			}
		}
	}
}

// ---------- Limits helpers ----------

// IsUnlimited reports whether a numeric limit value means "unlimited".
func IsUnlimited(v int64) bool { return v == 0 }

// IsUnlimitedInt reports whether an int limit value means "unlimited".
func IsUnlimitedInt(v int) bool { return v == 0 }

// CheckLimit returns nil if usage < limit, or an error describing the overage.
// If limit is 0 (unlimited), it always returns nil.
func CheckLimit(resource string, usage, limit int64) error {
	if limit == 0 {
		return nil // unlimited
	}
	if usage >= limit {
		return fmt.Errorf("%s limit reached: %d/%d", resource, usage, limit)
	}
	return nil
}

// CheckLimitInt is CheckLimit for int values.
func CheckLimitInt(resource string, usage, limit int) error {
	return CheckLimit(resource, int64(usage), int64(limit))
}

// ---------- JSON helpers ----------

// PlanFromJSON deserialises a Plan from JSON bytes.
func PlanFromJSON(data []byte) (*Plan, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("billing: unmarshal plan: %w", err)
	}
	if !ValidPlanID(p.ID) {
		return nil, fmt.Errorf("billing: invalid plan ID %q", p.ID)
	}
	return &p, nil
}

// PlansFromJSON deserialises a map of plans from JSON bytes.
func PlansFromJSON(data []byte) (map[PlanID]*Plan, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("billing: unmarshal plans: %w", err)
	}
	plans := make(map[PlanID]*Plan, len(raw))
	for k, v := range raw {
		id := PlanID(k)
		if !ValidPlanID(id) {
			return nil, fmt.Errorf("billing: invalid plan ID %q", k)
		}
		p, err := PlanFromJSON(v)
		if err != nil {
			return nil, err
		}
		plans[id] = p
	}
	return plans, nil
}
