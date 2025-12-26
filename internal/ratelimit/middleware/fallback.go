package middleware

import (
	"context"
	"log/slog"

	"credo/internal/ratelimit/config"
	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/service/requestlimit"
	"credo/internal/ratelimit/store/bucket"
)

// fallbackLimiter provides in-memory rate limiting when the primary limiter is unavailable.
// Used by the circuit breaker to maintain rate limiting during store outages.
type fallbackLimiter struct {
	requests *requestlimit.Service
}

// NewFallbackLimiter creates a fallback rate limiter that uses in-memory storage.
// Returns nil if cfg or allowlistStore is nil, logging an error if a logger is provided.
func NewFallbackLimiter(cfg *config.Config, allowlistStore requestlimit.AllowlistStore, logger *slog.Logger) RateLimiter {
	if cfg == nil || allowlistStore == nil {
		if logger != nil {
			logger.Error("fallback limiter requires config and allowlist store")
		}
		return nil
	}
	requests, err := requestlimit.New(
		bucket.New(),
		allowlistStore,
		requestlimit.WithLogger(logger),
		requestlimit.WithConfig(cfg),
	)
	if err != nil {
		if logger != nil {
			logger.Error("failed to initialize fallback rate limiter", "error", err)
		}
		return nil
	}
	return &fallbackLimiter{requests: requests}
}

func (f *fallbackLimiter) CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return f.requests.CheckIP(ctx, ip, class)
}

func (f *fallbackLimiter) CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return f.requests.CheckBoth(ctx, ip, userID, class)
}

func (f *fallbackLimiter) CheckGlobalThrottle(ctx context.Context) (bool, error) {
	// Fallback allows all traffic for global throttle during degraded mode
	return true, nil
}

// fallbackClientLimiter provides in-memory client rate limiting when the primary limiter is unavailable.
type fallbackClientLimiter struct {
	buckets *bucket.InMemoryBucketStore
	limit   config.Limit
}

// NewFallbackClientLimiter creates a fallback client rate limiter with in-memory storage.
// Returns nil if cfg is nil.
func NewFallbackClientLimiter(cfg *config.ClientLimitConfig) ClientRateLimiter {
	if cfg == nil {
		return nil
	}
	limit := cfg.PublicLimit
	return &fallbackClientLimiter{
		buckets: bucket.New(),
		limit:   limit,
	}
}

func (f *fallbackClientLimiter) Check(ctx context.Context, clientID, endpoint string) (*models.RateLimitResult, error) {
	return f.buckets.Allow(ctx, models.NewClientRateLimitKey(clientID, endpoint), f.limit.RequestsPerWindow, f.limit.Window)
}
