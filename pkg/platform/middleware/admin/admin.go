package admin

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	request "credo/pkg/platform/middleware/request"
)

func RequireAdminToken(expectedToken string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Admin-Token")
			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				ctx := r.Context()
				requestID := request.GetRequestID(ctx)
				logger.WarnContext(ctx, "admin token mismatch",
					"request_id", requestID,
				)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","error_description":"admin token required"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
