package testutil

import (
	"context"
	"net/http"

	authmw "credo/pkg/platform/middleware/auth"
)

// WithUserID adds a user ID to the request context.
// This simulates what the auth middleware would do for authenticated requests.
func WithUserID(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), authmw.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

// WithSessionID adds a session ID to the request context.
func WithSessionID(req *http.Request, sessionID string) *http.Request {
	ctx := context.WithValue(req.Context(), authmw.ContextKeySessionID, sessionID)
	return req.WithContext(ctx)
}

// WithAuth adds both user ID and session ID to the request context.
// This is the typical state for an authenticated request.
func WithAuth(req *http.Request, userID, sessionID string) *http.Request {
	ctx := req.Context()
	if userID != "" {
		ctx = context.WithValue(ctx, authmw.ContextKeyUserID, userID)
	}
	if sessionID != "" {
		ctx = context.WithValue(ctx, authmw.ContextKeySessionID, sessionID)
	}
	return req.WithContext(ctx)
}

// WithContextValue adds an arbitrary key-value pair to the request context.
func WithContextValue(req *http.Request, key, value any) *http.Request {
	ctx := context.WithValue(req.Context(), key, value)
	return req.WithContext(ctx)
}
