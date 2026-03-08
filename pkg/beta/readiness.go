package beta

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// ReadinessChecker runs a suite of checks to determine if the system is ready for beta launch.
type ReadinessChecker struct {
	mu     sync.RWMutex
	checks []registeredCheck
}

type registeredCheck struct {
	name     string
	category string
	critical bool
	fn       ReadinessCheckFunc
}

// NewReadinessChecker creates a new readiness checker.
func NewReadinessChecker() *ReadinessChecker {
	return &ReadinessChecker{}
}

// Register adds a check to the readiness suite.
func (rc *ReadinessChecker) Register(name, category string, critical bool, fn ReadinessCheckFunc) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.checks = append(rc.checks, registeredCheck{
		name:     name,
		category: category,
		critical: critical,
		fn:       fn,
	})
}

// Run executes all registered checks and returns a readiness report.
func (rc *ReadinessChecker) Run() *ReadinessReport {
	rc.mu.RLock()
	checks := make([]registeredCheck, len(rc.checks))
	copy(checks, rc.checks)
	rc.mu.RUnlock()

	report := &ReadinessReport{
		Timestamp: time.Now().UTC(),
		Checks:    make([]ReadinessCheck, 0, len(checks)),
	}

	for _, c := range checks {
		result := c.fn()
		result.Name = c.name
		result.Category = c.category
		result.Critical = c.critical
		report.Checks = append(report.Checks, result)
	}

	// Compute summary
	for _, c := range report.Checks {
		report.Summary.Total++
		switch c.Status {
		case CheckPass:
			report.Summary.Passed++
		case CheckFail:
			report.Summary.Failed++
			if c.Critical {
				report.Summary.Critical++
			}
		case CheckWarn:
			report.Summary.Warnings++
		}
	}

	report.Ready = report.Summary.Critical == 0 && report.Summary.Failed == 0
	return report
}

// --- Built-in Readiness Checks ---

// DatabaseCheck verifies database connectivity.
func DatabaseCheck(db *sql.DB) ReadinessCheckFunc {
	return func() ReadinessCheck {
		if db == nil {
			return ReadinessCheck{Status: CheckFail, Message: "no database configured"}
		}
		if err := db.Ping(); err != nil {
			return ReadinessCheck{Status: CheckFail, Message: fmt.Sprintf("database ping failed: %v", err)}
		}
		return ReadinessCheck{Status: CheckPass, Message: "database is responsive"}
	}
}

// EncryptionKeyCheck verifies that an encryption key is configured.
func EncryptionKeyCheck() ReadinessCheckFunc {
	return func() ReadinessCheck {
		key := os.Getenv("OPERATOR_ENCRYPTION_KEY")
		if key == "" {
			return ReadinessCheck{Status: CheckFail, Message: "OPERATOR_ENCRYPTION_KEY not set"}
		}
		if len(key) < 32 {
			return ReadinessCheck{Status: CheckWarn, Message: fmt.Sprintf("encryption key is short (%d chars, recommend ≥32)", len(key))}
		}
		return ReadinessCheck{Status: CheckPass, Message: "encryption key configured"}
	}
}

// JWTSecretCheck verifies that JWT signing secret is configured.
func JWTSecretCheck() ReadinessCheckFunc {
	return func() ReadinessCheck {
		secret := os.Getenv("OPERATOR_JWT_SECRET")
		if secret == "" {
			return ReadinessCheck{Status: CheckFail, Message: "OPERATOR_JWT_SECRET not set"}
		}
		if len(secret) < 32 {
			return ReadinessCheck{Status: CheckWarn, Message: fmt.Sprintf("JWT secret is short (%d chars, recommend ≥32)", len(secret))}
		}
		return ReadinessCheck{Status: CheckPass, Message: "JWT secret configured"}
	}
}

// StripeConfigCheck verifies Stripe configuration.
func StripeConfigCheck() ReadinessCheckFunc {
	return func() ReadinessCheck {
		sk := os.Getenv("STRIPE_SECRET_KEY")
		wh := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if sk == "" {
			return ReadinessCheck{Status: CheckWarn, Message: "STRIPE_SECRET_KEY not set (billing disabled)"}
		}
		if wh == "" {
			return ReadinessCheck{Status: CheckWarn, Message: "STRIPE_WEBHOOK_SECRET not set"}
		}
		return ReadinessCheck{Status: CheckPass, Message: "Stripe configured"}
	}
}

// HealthEndpointCheck verifies the health endpoint is responding.
func HealthEndpointCheck(baseURL string) ReadinessCheckFunc {
	return func() ReadinessCheck {
		if baseURL == "" {
			return ReadinessCheck{Status: CheckWarn, Message: "no health endpoint URL configured"}
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(baseURL + "/health")
		if err != nil {
			return ReadinessCheck{Status: CheckFail, Message: fmt.Sprintf("health endpoint unreachable: %v", err)}
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return ReadinessCheck{Status: CheckFail, Message: fmt.Sprintf("health endpoint returned %d", resp.StatusCode)}
		}
		return ReadinessCheck{Status: CheckPass, Message: "health endpoint responding"}
	}
}

// MetricsEndpointCheck verifies the metrics endpoint is responding.
func MetricsEndpointCheck(baseURL string) ReadinessCheckFunc {
	return func() ReadinessCheck {
		if baseURL == "" {
			return ReadinessCheck{Status: CheckWarn, Message: "no metrics endpoint URL configured"}
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(baseURL + "/metrics")
		if err != nil {
			return ReadinessCheck{Status: CheckFail, Message: fmt.Sprintf("metrics endpoint unreachable: %v", err)}
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return ReadinessCheck{Status: CheckFail, Message: fmt.Sprintf("metrics endpoint returned %d", resp.StatusCode)}
		}
		return ReadinessCheck{Status: CheckPass, Message: "metrics endpoint responding"}
	}
}

// LLMProviderCheck verifies at least one LLM provider API key is configured.
func LLMProviderCheck() ReadinessCheckFunc {
	return func() ReadinessCheck {
		providers := []string{
			"OPENAI_API_KEY",
			"ANTHROPIC_API_KEY",
			"GEMINI_API_KEY",
			"GOOGLE_API_KEY",
		}
		var found []string
		for _, p := range providers {
			if os.Getenv(p) != "" {
				found = append(found, p)
			}
		}
		if len(found) == 0 {
			return ReadinessCheck{Status: CheckFail, Message: "no LLM provider API keys configured"}
		}
		return ReadinessCheck{Status: CheckPass, Message: fmt.Sprintf("%d LLM provider(s) configured", len(found))}
	}
}

// MinUsersCheck verifies a minimum number of users exist (for beta validation).
func MinUsersCheck(db *sql.DB, minCount int) ReadinessCheckFunc {
	return func() ReadinessCheck {
		if db == nil {
			return ReadinessCheck{Status: CheckFail, Message: "no database configured"}
		}
		var count int64
		err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&count)
		if err != nil {
			// Table might not exist yet, which is OK for readiness check
			return ReadinessCheck{Status: CheckWarn, Message: fmt.Sprintf("could not query users: %v", err)}
		}
		if count < int64(minCount) {
			return ReadinessCheck{Status: CheckWarn, Message: fmt.Sprintf("%d active users (target: %d)", count, minCount)}
		}
		return ReadinessCheck{Status: CheckPass, Message: fmt.Sprintf("%d active users", count)}
	}
}

// RegisterDefaultChecks registers a standard set of readiness checks.
func RegisterDefaultChecks(rc *ReadinessChecker, db *sql.DB, healthURL string) {
	rc.Register("database", "database", true, DatabaseCheck(db))
	rc.Register("encryption_key", "security", true, EncryptionKeyCheck())
	rc.Register("jwt_secret", "security", true, JWTSecretCheck())
	rc.Register("stripe", "billing", false, StripeConfigCheck())
	rc.Register("llm_provider", "integrations", true, LLMProviderCheck())
	if healthURL != "" {
		rc.Register("health_endpoint", "monitoring", false, HealthEndpointCheck(healthURL))
		rc.Register("metrics_endpoint", "monitoring", false, MetricsEndpointCheck(healthURL))
	}
}
