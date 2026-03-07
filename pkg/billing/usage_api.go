package billing

import (
	"net/http"
	"strconv"
	"time"
)

// UsageAPI provides HTTP handlers for the usage dashboard.
type UsageAPI struct {
	usageStore UsageStore
	subStore   SubscriptionStore
	catalogue  *Catalogue
}

// NewUsageAPI creates a UsageAPI with the given stores.
func NewUsageAPI(usageStore UsageStore, subStore SubscriptionStore, catalogue *Catalogue) *UsageAPI {
	return &UsageAPI{
		usageStore: usageStore,
		subStore:   subStore,
		catalogue:  catalogue,
	}
}

// RegisterRoutes registers usage dashboard routes on the given mux.
func (a *UsageAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/billing/usage", a.handleGetUsageSummary)
	mux.HandleFunc("GET /api/v1/billing/usage/daily", a.handleGetDailyUsage)
	mux.HandleFunc("GET /api/v1/billing/usage/models", a.handleGetModelUsage)
	mux.HandleFunc("GET /api/v1/billing/usage/events", a.handleListUsageEvents)
	mux.HandleFunc("GET /api/v1/billing/usage/limits", a.handleGetUsageLimits)
}

// handleGetUsageSummary returns the aggregate usage for the current billing period.
func (a *UsageAPI) handleGetUsageSummary(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	if a.usageStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "usage tracking not configured",
			"code":  "not_configured",
		})
		return
	}

	since, until := a.parsePeriod(r, userID)

	summary, err := a.usageStore.GetSummary(userID, since, until)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to get usage summary",
			"code":  "internal_error",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"summary":      summary,
		"period_start": since.Format(time.RFC3339),
		"period_end":   until.Format(time.RFC3339),
	})
}

// handleGetDailyUsage returns per-day usage breakdown.
func (a *UsageAPI) handleGetDailyUsage(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	if a.usageStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "usage tracking not configured",
			"code":  "not_configured",
		})
		return
	}

	since, until := a.parseTimeRange(r, 30) // default 30 days

	daily, err := a.usageStore.GetDaily(userID, since, until)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to get daily usage",
			"code":  "internal_error",
		})
		return
	}

	if daily == nil {
		daily = []*DailyUsage{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"daily":        daily,
		"count":        len(daily),
		"period_start": since.Format(time.RFC3339),
		"period_end":   until.Format(time.RFC3339),
	})
}

// handleGetModelUsage returns per-model usage breakdown.
func (a *UsageAPI) handleGetModelUsage(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	if a.usageStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "usage tracking not configured",
			"code":  "not_configured",
		})
		return
	}

	since, until := a.parseTimeRange(r, 30)

	models, err := a.usageStore.GetByModel(userID, since, until)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to get model usage",
			"code":  "internal_error",
		})
		return
	}

	if models == nil {
		models = []*ModelUsage{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"models":       models,
		"count":        len(models),
		"period_start": since.Format(time.RFC3339),
		"period_end":   until.Format(time.RFC3339),
	})
}

// handleListUsageEvents returns paginated raw usage events.
func (a *UsageAPI) handleListUsageEvents(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	if a.usageStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "usage tracking not configured",
			"code":  "not_configured",
		})
		return
	}

	query := UsageQuery{
		UserID: userID,
		Model:  r.URL.Query().Get("model"),
	}

	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			query.Since = t
		}
	}
	if until := r.URL.Query().Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			query.Until = t
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil {
			query.Limit = n
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil {
			query.Offset = n
		}
	}

	events, err := a.usageStore.ListEvents(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to list usage events",
			"code":  "internal_error",
		})
		return
	}

	if events == nil {
		events = []*UsageEvent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"count":  len(events),
		"limit":  query.Limit,
		"offset": query.Offset,
	})
}

// handleGetUsageLimits returns the user's current usage against plan limits.
func (a *UsageAPI) handleGetUsageLimits(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authentication required",
			"code":  "unauthorized",
		})
		return
	}

	if a.usageStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "usage tracking not configured",
			"code":  "not_configured",
		})
		return
	}

	// Determine the user's plan.
	plan := a.catalogue.Get(PlanFree) // default to free
	periodStart := beginningOfMonth(time.Now().UTC())

	if a.subStore != nil {
		sub, err := a.subStore.GetByUserID(userID)
		if err == nil && sub != nil && sub.IsActive() {
			if p := a.catalogue.Get(sub.PlanID); p != nil {
				plan = p
			}
			if !sub.CurrentPeriodStart.IsZero() {
				periodStart = sub.CurrentPeriodStart
			}
		}
	}

	// Get current period usage.
	tokenUsage, err := a.usageStore.GetCurrentPeriodUsage(userID, periodStart)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to get token usage",
			"code":  "internal_error",
		})
		return
	}

	messageUsage, err := a.usageStore.GetCurrentPeriodMessages(userID, periodStart)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "failed to get message usage",
			"code":  "internal_error",
		})
		return
	}

	limits := map[string]any{
		"plan_id":      plan.ID,
		"plan_name":    plan.Name,
		"period_start": periodStart.Format(time.RFC3339),
		"tokens": map[string]any{
			"used":      tokenUsage,
			"limit":     plan.Limits.MaxTokensPerMonth,
			"unlimited": IsUnlimited(plan.Limits.MaxTokensPerMonth),
		},
		"messages": map[string]any{
			"used":      messageUsage,
			"limit":     plan.Limits.MaxMessagesPerMonth,
			"unlimited": IsUnlimited(plan.Limits.MaxMessagesPerMonth),
		},
	}

	writeJSON(w, http.StatusOK, limits)
}

// parsePeriod determines the billing period from the user's subscription,
// or falls back to the current calendar month.
func (a *UsageAPI) parsePeriod(r *http.Request, userID string) (time.Time, time.Time) {
	now := time.Now().UTC()

	// Check for explicit query params first.
	if since := r.URL.Query().Get("since"); since != "" {
		if until := r.URL.Query().Get("until"); until != "" {
			s, errS := time.Parse(time.RFC3339, since)
			u, errU := time.Parse(time.RFC3339, until)
			if errS == nil && errU == nil {
				return s, u
			}
		}
	}

	// Try subscription period.
	if a.subStore != nil {
		sub, err := a.subStore.GetByUserID(userID)
		if err == nil && sub != nil && sub.IsActive() && !sub.CurrentPeriodStart.IsZero() {
			return sub.CurrentPeriodStart, sub.CurrentPeriodEnd
		}
	}

	// Fallback to current calendar month.
	start := beginningOfMonth(now)
	end := start.AddDate(0, 1, 0)
	return start, end
}

// parseTimeRange extracts since/until from query params with a default lookback in days.
func (a *UsageAPI) parseTimeRange(r *http.Request, defaultDays int) (time.Time, time.Time) {
	now := time.Now().UTC()
	since := now.AddDate(0, 0, -defaultDays)
	until := now

	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = t
		}
	}

	// Also check "days" param as shorthand.
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			since = now.AddDate(0, 0, -n)
		}
	}

	return since, until
}

// beginningOfMonth returns midnight on the first day of the given time's month.
func beginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
