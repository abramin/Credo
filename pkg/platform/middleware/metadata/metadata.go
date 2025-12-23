package metadata

import (
	"context"
	"net/http"
	"strings"
)

// Context keys for client metadata.
type contextKeyClientIP struct{}
type contextKeyUserAgent struct{}

// ClientMetadata extracts client IP address and User-Agent from the request
// and adds them to the context for use by handlers and services.
// This middleware should be applied early in the chain.
func ClientMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIPFromRequest(r)
		userAgent := r.Header.Get("User-Agent")

		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyClientIP{}, ip)
		ctx = context.WithValue(ctx, contextKeyUserAgent{}, userAgent)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientIP retrieves the client IP address from the context.
func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(contextKeyClientIP{}).(string); ok {
		return ip
	}
	return ""
}

// GetUserAgent retrieves the User-Agent from the context.
func GetUserAgent(ctx context.Context) string {
	if ua, ok := ctx.Value(contextKeyUserAgent{}).(string); ok {
		return ua
	}
	return ""
}

// WithClientMetadata injects client IP and User-Agent into a context.
// Useful for service unit tests that don't run the full HTTP middleware chain.
func WithClientMetadata(ctx context.Context, clientIP, userAgent string) context.Context {
	ctx = context.WithValue(ctx, contextKeyClientIP{}, clientIP)
	ctx = context.WithValue(ctx, contextKeyUserAgent{}, userAgent)
	return ctx
}

// ClientIPFromRequest extracts the real client IP from the request, handling proxies and load balancers.
func ClientIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header first (standard for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs (client, proxy1, proxy2, ...)
		// Take the first IP which is the original client
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header (used by nginx and other proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (direct connection)
	// RemoteAddr is in format "ip:port", so we need to strip the port
	if addr := r.RemoteAddr; addr != "" {
		// For IPv6, format is [::1]:port
		// For IPv4, format is 127.0.0.1:port
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			return addr[:idx]
		}
		return addr
	}

	return "unknown"
}
