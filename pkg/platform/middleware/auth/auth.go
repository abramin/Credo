package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	request "credo/pkg/platform/middleware/request"
)

// JWTValidator defines the interface for validating JWT tokens
type JWTValidator interface {
	ValidateToken(tokenString string) (*JWTClaims, error)
}

// TokenRevocationChecker defines the interface for checking if tokens are revoked
type TokenRevocationChecker interface {
	IsTokenRevoked(ctx context.Context, jti string) (bool, error)
}

// JWTClaims represents the claims we expect from the JWT validator
type JWTClaims struct {
	UserID    string
	SessionID string
	ClientID  string
	JTI       string // JWT ID for revocation tracking
}

// Context keys for storing authenticated user information
type contextKeyUserID struct{}
type contextKeySessionID struct{}
type contextKeyClientID struct{}

// ContextKeyUserID is exported for use in handlers
var (
	ContextKeyUserID    = contextKeyUserID{}
	ContextKeySessionID = contextKeySessionID{}
	ContextKeyClientID  = contextKeyClientID{}
)

// GetUserID retrieves the authenticated user ID from the context
func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetSessionID retrieves the session ID from the context
func GetSessionID(ctx context.Context) string {
	sessionID, ok := ctx.Value(ContextKeySessionID).(string)
	if !ok {
		return ""
	}
	return sessionID
}

func GetClientID(ctx context.Context) string {
	clientID, ok := ctx.Value(ContextKeyClientID).(string)
	if !ok {
		return ""
	}
	return clientID
}

// writeJSONError writes a JSON error response with the given status code and error details.
func writeJSONError(w http.ResponseWriter, status int, errCode, errDesc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(fmt.Appendf(nil, `{"error":"%s","error_description":"%s"}`, errCode, errDesc))
}

func RequireAuth(validator JWTValidator, revocationChecker TokenRevocationChecker, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			const bearerPrefix = "Bearer "
			if after, ok := strings.CutPrefix(authHeader, bearerPrefix); ok {
				token := after
				claims, err := validator.ValidateToken(token)
				if err != nil {
					ctx := r.Context()
					requestID := request.GetRequestID(ctx)
					logger.WarnContext(ctx, "unauthorized access - invalid token",
						"error", err,
						"request_id", requestID,
					)
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
					return
				}

				ctx := r.Context()

				// TR-4: Middleware revocation check (PRD-016).
				if revocationChecker != nil {
					if claims.JTI == "" {
						requestID := request.GetRequestID(ctx)
						logger.WarnContext(ctx, "unauthorized access - missing token jti",
							"request_id", requestID,
						)
						writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
						return
					}

					revoked, err := revocationChecker.IsTokenRevoked(ctx, claims.JTI)
					if err != nil {
						requestID := request.GetRequestID(ctx)
						logger.ErrorContext(ctx, "failed to check token revocation",
							"error", err,
							"request_id", requestID,
						)
						writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to validate token")
						return
					}
					if revoked {
						requestID := request.GetRequestID(ctx)
						logger.WarnContext(ctx, "unauthorized access - token revoked",
							"jti", claims.JTI,
							"request_id", requestID,
						)
						writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Token has been revoked")
						return
					}
				}

				ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
				ctx = context.WithValue(ctx, ContextKeySessionID, claims.SessionID)
				ctx = context.WithValue(ctx, ContextKeyClientID, claims.ClientID)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No Authorization header or invalid format
			ctx := r.Context()
			requestID := request.GetRequestID(ctx)
			logger.WarnContext(ctx, "unauthorized access - missing token",
				"request_id", requestID,
			)
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid Authorization header")
		})
	}
}
