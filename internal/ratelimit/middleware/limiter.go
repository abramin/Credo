package middleware

import (
	"context"

	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/service/globalthrottle"
	"credo/internal/ratelimit/service/requestlimit"
)

// Limiter implements RateLimiter by composing the focused rate limiting services.
// This replaces the checker.Service facade with a simpler composition that only
// exposes what the middleware actually needs.
type Limiter struct {
	requests       *requestlimit.Service
	globalThrottle *globalthrottle.Service
}

// NewLimiter creates a Limiter that composes request limiting and global throttle services.
func NewLimiter(requests *requestlimit.Service, globalThrottle *globalthrottle.Service) *Limiter {
	return &Limiter{
		requests:       requests,
		globalThrottle: globalThrottle,
	}
}

func (l *Limiter) CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return l.requests.CheckIP(ctx, ip, class)
}

func (l *Limiter) CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return l.requests.CheckBoth(ctx, ip, userID, class)
}

func (l *Limiter) CheckGlobalThrottle(ctx context.Context) (bool, error) {
	return l.globalThrottle.Check(ctx)
}
