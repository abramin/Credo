package version

import (
	"log/slog"
	"net/http"

	id "credo/pkg/domain"
	"credo/pkg/requestcontext"
)

// ValidateTokenVersion creates middleware that validates the token's API version
// against the route's API version.
//
// Forward compatibility rules (v1 tokens work on v2 routes):
//   - routeVersion.IsAtLeast(tokenVersion) must be true
//   - v1 token on v2 route: OK (route v2 >= token v1)
//   - v2 token on v1 route: REJECTED (route v1 < token v2)
//
// This middleware must run AFTER:
//  1. ExtractVersion middleware (sets route version in context)
//  2. Auth middleware (sets token version in context)
//
// Usage:
//
//	v1.Group(func(r chi.Router) {
//	    r.Use(auth.RequireAuth(...))
//	    r.Use(version.ValidateTokenVersion(logger))
//	    // ... protected routes
//	})
func ValidateTokenVersion(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get route version from context (set by ExtractVersion middleware)
			routeVersion := requestcontext.APIVersion(ctx)
			if routeVersion.IsNil() {
				// Should not happen if ExtractVersion ran first
				logger.ErrorContext(ctx, "version validation failed: route version not set",
					"request_id", requestcontext.RequestID(ctx),
				)
				writeVersionError(w, http.StatusInternalServerError, "server_error", "route version not configured")
				return
			}

			// Get token version from context (set by auth middleware)
			tokenVersion := requestcontext.TokenAPIVersion(ctx)
			if tokenVersion.IsNil() {
				// Token has no version claim - treat as v1 (legacy tokens)
				tokenVersion = id.APIVersionV1
			}

			// Forward compatibility check:
			// Route version must be >= token version
			// This allows v1 tokens to work on v2 routes (upgrade path)
			// but rejects v2 tokens on v1 routes (prevents replay attacks)
			if !routeVersion.IsAtLeast(tokenVersion) {
				logger.WarnContext(ctx, "cross-version token replay rejected",
					"token_version", tokenVersion.String(),
					"route_version", routeVersion.String(),
					"request_id", requestcontext.RequestID(ctx),
					"user_id", requestcontext.UserID(ctx).String(),
				)
				writeVersionError(w, http.StatusForbidden, "invalid_token",
					"token API version not compatible with this endpoint version")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
