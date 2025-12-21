package device

import "context"

type contextKeyDeviceID struct{}
type contextKeyDeviceFingerprint struct{}

// GetDeviceID retrieves the device identifier (cookie value) from the context.
func GetDeviceID(ctx context.Context) string {
	if deviceID, ok := ctx.Value(contextKeyDeviceID{}).(string); ok {
		return deviceID
	}
	return ""
}

// WithDeviceID injects a device identifier into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, contextKeyDeviceID{}, deviceID)
}

// GetDeviceFingerprint retrieves the pre-computed device fingerprint from the context.
func GetDeviceFingerprint(ctx context.Context) string {
	if fp, ok := ctx.Value(contextKeyDeviceFingerprint{}).(string); ok {
		return fp
	}
	return ""
}

// WithDeviceFingerprint injects a device fingerprint into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithDeviceFingerprint(ctx context.Context, fingerprint string) context.Context {
	return context.WithValue(ctx, contextKeyDeviceFingerprint{}, fingerprint)
}
