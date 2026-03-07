package users

import (
	"context"
	"net/http"
	"strings"
)

// Context keys for authenticated user information.
type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyEmail  contextKey = "email"
	contextKeyClaims contextKey = "claims"
)

// ContextKeyUserID returns the context key used for the authenticated user ID.
// This is exported for use by other packages that need to inject user context
// (e.g., in tests or inter-package middleware).
func ContextKeyUserID() contextKey { return contextKeyUserID }

// AuthMiddleware returns an HTTP middleware that validates JWT access tokens.
// Protected routes receive user info via request context.
func AuthMiddleware(ts *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				writeError(w, http.StatusUnauthorized, "missing_token", "Authorization header with Bearer token is required")
				return
			}

			claims, err := ts.ValidateAccessToken(tokenStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired access token")
				return
			}

			// Inject claims into request context.
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, contextKeyClaims, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user's ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyUserID).(string); ok {
		return v
	}
	return ""
}

// EmailFromContext returns the authenticated user's email from the request context.
func EmailFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyEmail).(string); ok {
		return v
	}
	return ""
}

// ClaimsFromContext returns the full token claims from the request context.
func ClaimsFromContext(ctx context.Context) *TokenClaims {
	if v, ok := ctx.Value(contextKeyClaims).(*TokenClaims); ok {
		return v
	}
	return nil
}

// extractBearerToken extracts the token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
