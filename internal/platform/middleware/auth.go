package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// JWTValidator defines the interface for validating JWT tokens
type JWTValidator interface {
	ValidateToken(tokenString string) (*JWTClaims, error)
}

// JWTClaims represents the claims we expect from the JWT validator
type JWTClaims struct {
	UserID    string
	SessionID string
	ClientID  string
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

func RequireAuth(validator JWTValidator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			const bearerPrefix = "Bearer "
			if after, ok := strings.CutPrefix(authHeader, bearerPrefix); ok {
				token := after
				claims, err := validator.ValidateToken(token)
				if err != nil {
					ctx := r.Context()
					requestID := GetRequestID(ctx)
					logger.WarnContext(ctx, "unauthorized access - invalid token",
						"error", err,
						"request_id", requestID,
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, err = w.Write([]byte(`{"error":"unauthorized","error_description":"Invalid or expired token"}`))
					if err != nil {
						logger.ErrorContext(ctx, "failed to write unauthorized response",
							"error", err,
							"request_id", requestID,
						)
					}
					return
				}

				ctx := r.Context()
				ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
				ctx = context.WithValue(ctx, ContextKeySessionID, claims.SessionID)
				ctx = context.WithValue(ctx, ContextKeyClientID, claims.ClientID)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No Authorization header or invalid format
			ctx := r.Context()
			requestID := GetRequestID(ctx)
			logger.WarnContext(ctx, "unauthorized access - missing token",
				"request_id", requestID,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(`{"error":"unauthorized","error_description":"Missing or invalid Authorization header"}`))
			if err != nil {
				logger.ErrorContext(ctx, "failed to write unauthorized response",
					"error", err,
					"request_id", requestID,
				)
			}
		})
	}
}
