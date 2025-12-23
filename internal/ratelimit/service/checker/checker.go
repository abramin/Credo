package checker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/service/authlockout"
	"credo/internal/ratelimit/service/globalthrottle"
	"credo/internal/ratelimit/service/quota"
	"credo/internal/ratelimit/service/requestlimit"
	id "credo/pkg/domain"
)

// Service is a facade composing focused rate limiting services.
// Middleware depends on this unified interface.
type Service struct {
	requests       *requestlimit.Service
	authLockout    *authlockout.Service
	quotas         *quota.Service
	globalThrottle *globalthrottle.Service
	logger         *slog.Logger
}

type Option func(*Service)

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

func New(
	requests *requestlimit.Service,
	authLockout *authlockout.Service,
	quotas *quota.Service,
	globalThrottle *globalthrottle.Service,
	opts ...Option,
) (*Service, error) {
	if requests == nil {
		return nil, fmt.Errorf("requests service is required")
	}
	if authLockout == nil {
		return nil, fmt.Errorf("auth lockout service is required")
	}
	if quotas == nil {
		return nil, fmt.Errorf("quotas service is required")
	}
	if globalThrottle == nil {
		return nil, fmt.Errorf("global throttle service is required")
	}

	svc := &Service{
		requests:       requests,
		authLockout:    authLockout,
		quotas:         quotas,
		globalThrottle: globalThrottle,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (s *Service) CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return s.requests.CheckIP(ctx, ip, class)
}

func (s *Service) CheckUserRateLimit(ctx context.Context, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return s.requests.CheckUser(ctx, userID, class)
}

func (s *Service) CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	return s.requests.CheckBoth(ctx, ip, userID, class)
}

func (s *Service) CheckAuthRateLimit(ctx context.Context, identifier, ip string) (*models.AuthRateLimitResult, error) {
	result, err := s.authLockout.Check(ctx, identifier, ip)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Secondary defense: IP rate limit for auth endpoints
	ipResult, err := s.requests.CheckIP(ctx, ip, models.ClassAuth)
	if err != nil {
		return nil, err
	}
	if !ipResult.Allowed {
		return &models.AuthRateLimitResult{
			RateLimitResult: *ipResult,
			RequiresCaptcha: result.RequiresCaptcha,
			FailureCount:    result.FailureCount,
		}, nil
	}

	// Update with IP rate limit info if auth check passed
	result.RateLimitResult.Remaining = ipResult.Remaining
	result.RateLimitResult.ResetAt = ipResult.ResetAt

	return result, nil
}

func (s *Service) RecordAuthFailure(ctx context.Context, identifier, ip string) (*models.AuthLockout, error) {
	return s.authLockout.RecordFailure(ctx, identifier, ip)
}

func (s *Service) ClearAuthFailures(ctx context.Context, identifier, ip string) error {
	return s.authLockout.Clear(ctx, identifier, ip)
}

func (s *Service) GetProgressiveBackoff(failureCount int) time.Duration {
	return s.authLockout.GetProgressiveBackoff(failureCount)
}

func (s *Service) CheckAPIKeyQuota(ctx context.Context, apiKeyID id.APIKeyID) (*models.APIKeyQuota, error) {
	return s.quotas.Check(ctx, apiKeyID)
}

func (s *Service) CheckGlobalThrottle(ctx context.Context) (bool, error) {
	return s.globalThrottle.Check(ctx)
}
