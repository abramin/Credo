package testutil

import (
	"context"
	"net/http"

	id "credo/pkg/domain"
	"credo/pkg/requestcontext"
)

// WithUserID adds a user ID to the request context.
// This simulates what the auth middleware would do for authenticated requests.
// If the userID is not a valid UUID, it will not be added to the context.
func WithUserID(req *http.Request, userID string) *http.Request {
	if parsedUserID, err := id.ParseUserID(userID); err == nil {
		ctx := requestcontext.WithUserID(req.Context(), parsedUserID)
		return req.WithContext(ctx)
	}
	return req
}

// WithSessionID adds a session ID to the request context.
// If the sessionID is not a valid UUID, it will not be added to the context.
func WithSessionID(req *http.Request, sessionID string) *http.Request {
	if parsedSessionID, err := id.ParseSessionID(sessionID); err == nil {
		ctx := requestcontext.WithSessionID(req.Context(), parsedSessionID)
		return req.WithContext(ctx)
	}
	return req
}

// WithAuth adds both user ID and session ID to the request context.
// This is the typical state for an authenticated request.
// Invalid IDs are silently ignored.
func WithAuth(req *http.Request, userID, sessionID string) *http.Request {
	ctx := req.Context()
	if userID != "" {
		if parsedUserID, err := id.ParseUserID(userID); err == nil {
			ctx = requestcontext.WithUserID(ctx, parsedUserID)
		}
	}
	if sessionID != "" {
		if parsedSessionID, err := id.ParseSessionID(sessionID); err == nil {
			ctx = requestcontext.WithSessionID(ctx, parsedSessionID)
		}
	}
	return req.WithContext(ctx)
}

// WithContextValue adds an arbitrary key-value pair to the request context.
func WithContextValue(req *http.Request, key, value any) *http.Request {
	ctx := context.WithValue(req.Context(), key, value)
	return req.WithContext(ctx)
}
