// Package requestcontext provides HTTP-independent context accessors for request-scoped values.
//
// This package defines context keys and getter/setter functions for values that are
// typically set by middleware but consumed by services. By keeping this package free
// of net/http dependencies, services can import only what they need without pulling
// in HTTP-related code.
//
// Usage in services (read values):
//
//	userID := requestcontext.UserID(ctx)
//	requestID := requestcontext.RequestID(ctx)
//	now := requestcontext.Now(ctx)
//
// Usage in middleware (set values):
//
//	ctx = requestcontext.WithUserID(ctx, userID)
//	ctx = requestcontext.WithRequestID(ctx, requestID)
//
// Usage in tests (inject values):
//
//	ctx = requestcontext.WithTime(ctx, fixedTime)
//	ctx = requestcontext.WithDeviceFingerprint(ctx, "fingerprint-hash")
package requestcontext

import (
	"context"
	"time"

	id "credo/pkg/domain"
)

// Context key types (unexported for encapsulation).
type (
	userIDKey            struct{}
	sessionIDKey         struct{}
	clientIDKey          struct{}
	deviceIDKey          struct{}
	deviceFingerprintKey struct{}
	clientIPKey          struct{}
	userAgentKey         struct{}
	requestIDKey         struct{}
	requestTimeKey       struct{}
)

// Exported context keys for direct use in tests that need context.WithValue.
var (
	ContextKeyUserID            = userIDKey{}
	ContextKeySessionID         = sessionIDKey{}
	ContextKeyClientID          = clientIDKey{}
	ContextKeyDeviceID          = deviceIDKey{}
	ContextKeyDeviceFingerprint = deviceFingerprintKey{}
	ContextKeyClientIP          = clientIPKey{}
	ContextKeyUserAgent         = userAgentKey{}
	ContextKeyRequestID         = requestIDKey{}
	ContextKeyRequestTime       = requestTimeKey{}
)

// -----------------------------------------------------------------------------
// Auth context (user, session, client IDs)
// -----------------------------------------------------------------------------

// UserID retrieves the authenticated user ID from the context.
// Returns the zero value (nil UUID) if not set.
func UserID(ctx context.Context) id.UserID {
	if userID, ok := ctx.Value(ContextKeyUserID).(id.UserID); ok {
		return userID
	}
	return id.UserID{}
}

// WithUserID injects a user ID into the context.
func WithUserID(ctx context.Context, userID id.UserID) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// SessionID retrieves the session ID from the context.
// Returns the zero value (nil UUID) if not set.
func SessionID(ctx context.Context) id.SessionID {
	if sessionID, ok := ctx.Value(ContextKeySessionID).(id.SessionID); ok {
		return sessionID
	}
	return id.SessionID{}
}

// WithSessionID injects a session ID into the context.
func WithSessionID(ctx context.Context, sessionID id.SessionID) context.Context {
	return context.WithValue(ctx, ContextKeySessionID, sessionID)
}

// ClientID retrieves the client ID from the context.
// Returns the zero value (nil UUID) if not set.
func ClientID(ctx context.Context) id.ClientID {
	if clientID, ok := ctx.Value(ContextKeyClientID).(id.ClientID); ok {
		return clientID
	}
	return id.ClientID{}
}

// WithClientID injects a client ID into the context.
func WithClientID(ctx context.Context, clientID id.ClientID) context.Context {
	return context.WithValue(ctx, ContextKeyClientID, clientID)
}

// -----------------------------------------------------------------------------
// Device context
// -----------------------------------------------------------------------------

// DeviceID retrieves the device identifier (cookie value) from the context.
func DeviceID(ctx context.Context) string {
	if deviceID, ok := ctx.Value(ContextKeyDeviceID).(string); ok {
		return deviceID
	}
	return ""
}

// WithDeviceID injects a device identifier into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, ContextKeyDeviceID, deviceID)
}

// DeviceFingerprint retrieves the pre-computed device fingerprint from the context.
func DeviceFingerprint(ctx context.Context) string {
	if fp, ok := ctx.Value(ContextKeyDeviceFingerprint).(string); ok {
		return fp
	}
	return ""
}

// WithDeviceFingerprint injects a device fingerprint into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithDeviceFingerprint(ctx context.Context, fingerprint string) context.Context {
	return context.WithValue(ctx, ContextKeyDeviceFingerprint, fingerprint)
}

// -----------------------------------------------------------------------------
// Client metadata (IP, User-Agent)
// -----------------------------------------------------------------------------

// ClientIP retrieves the client IP address from the context.
func ClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(ContextKeyClientIP).(string); ok {
		return ip
	}
	return ""
}

// UserAgent retrieves the User-Agent from the context.
func UserAgent(ctx context.Context) string {
	if ua, ok := ctx.Value(ContextKeyUserAgent).(string); ok {
		return ua
	}
	return ""
}

// WithClientMetadata injects client IP and User-Agent into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithClientMetadata(ctx context.Context, clientIP, userAgent string) context.Context {
	ctx = context.WithValue(ctx, ContextKeyClientIP, clientIP)
	ctx = context.WithValue(ctx, ContextKeyUserAgent, userAgent)
	return ctx
}

// -----------------------------------------------------------------------------
// Request metadata
// -----------------------------------------------------------------------------

// RequestID retrieves the request ID from the context.
func RequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return reqID
	}
	return ""
}

// WithRequestID injects a request ID into the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// -----------------------------------------------------------------------------
// Request time
// -----------------------------------------------------------------------------

// Now retrieves the request-scoped time from context.
// Falls back to time.Now() if not set (for non-HTTP contexts like workers, CLI, tests).
func Now(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ContextKeyRequestTime).(time.Time); ok {
		return t
	}
	return time.Now()
}

// WithTime injects a specific time into a context.
// Useful for:
//   - Service unit tests that don't run the full HTTP middleware chain
//   - Workers that need consistent time within a batch operation
//   - CLI commands
func WithTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ContextKeyRequestTime, t)
}
