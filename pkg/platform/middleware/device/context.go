package device

import (
	"context"

	"credo/pkg/requestcontext"
)

// GetDeviceID retrieves the device identifier (cookie value) from the context.
// Deprecated: Use requestcontext.DeviceID(ctx) instead.
func GetDeviceID(ctx context.Context) string {
	return requestcontext.DeviceID(ctx)
}

// WithDeviceID injects a device identifier into a context.
// Deprecated: Use requestcontext.WithDeviceID(ctx, deviceID) instead.
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return requestcontext.WithDeviceID(ctx, deviceID)
}

// GetDeviceFingerprint retrieves the pre-computed device fingerprint from the context.
// Deprecated: Use requestcontext.DeviceFingerprint(ctx) instead.
func GetDeviceFingerprint(ctx context.Context) string {
	return requestcontext.DeviceFingerprint(ctx)
}

// WithDeviceFingerprint injects a device fingerprint into a context.
// Deprecated: Use requestcontext.WithDeviceFingerprint(ctx, fingerprint) instead.
func WithDeviceFingerprint(ctx context.Context, fingerprint string) context.Context {
	return requestcontext.WithDeviceFingerprint(ctx, fingerprint)
}
