package authlockout

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"credo/internal/ratelimit/config"
	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/ports"
	dErrors "credo/pkg/domain-errors"
	requesttime "credo/pkg/platform/middleware/requesttime"
	"credo/pkg/platform/privacy"
)

// Store is a subset of ports.AuthLockoutStore (excludes cleanup methods).
type Store interface {
	RecordFailure(ctx context.Context, identifier string) (*models.AuthLockout, error)
	Get(ctx context.Context, identifier string) (*models.AuthLockout, error)
	Clear(ctx context.Context, identifier string) error
	IsLocked(ctx context.Context, identifier string) (bool, *time.Time, error)
	Update(ctx context.Context, record *models.AuthLockout) error
}

// AuditPublisher is an alias to the shared interface.
type AuditPublisher = ports.AuditPublisher

type Service struct {
	store          Store
	auditPublisher AuditPublisher
	logger         *slog.Logger
	config         *config.AuthLockoutConfig
}

type Option func(*Service)

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

func WithAuditPublisher(publisher AuditPublisher) Option {
	return func(s *Service) {
		s.auditPublisher = publisher
	}
}

func WithConfig(cfg *config.AuthLockoutConfig) Option {
	return func(s *Service) {
		s.config = cfg
	}
}

func New(store Store, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("auth lockout store is required")
	}

	defaultCfg := config.DefaultConfig().AuthLockout
	svc := &Service{
		store:  store,
		config: &defaultCfg,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (s *Service) Check(ctx context.Context, identifier, ip string) (*models.AuthRateLimitResult, error) {
	key := models.NewAuthLockoutKey(identifier, ip).String()
	failureRecord, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to get auth lockout record")
	}

	// Use zero-valued record for consistent code path (prevents timing-based enumeration).
	// All checks execute regardless of record existence to ensure constant-time behavior.
	record := failureRecord
	if record == nil {
		record = &models.AuthLockout{}
	}

	now := requesttime.Now(ctx)

	// Check if currently hard-locked (FR-2b: "hard lock for 15 minutes")
	if record.IsLockedAt(now) {
		retryAfter := max(int(record.LockedUntil.Sub(now).Seconds()), 0)
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "auth_lockout_triggered",
			"identifier", identifier,
			"ip", privacy.AnonymizeIP(ip),
			"locked_until", record.LockedUntil,
		)
		return &models.AuthRateLimitResult{
			RateLimitResult: models.RateLimitResult{
				Allowed:    false,
				ResetAt:    *record.LockedUntil,
				RetryAfter: retryAfter,
			},
			RequiresCaptcha: record.RequiresCaptcha,
			FailureCount:    record.FailureCount,
		}, nil
	}

	// Check failure count against sliding window (FR-2b: "5 attempts/15 min")
	if record.IsAttemptLimitReached(s.config.AttemptsPerWindow) {
		// Block - too many attempts in window
		resetAt := s.config.ResetTime(record.LastFailureAt)
		retryAfter := max(int(resetAt.Sub(now).Seconds()), 0)
		return &models.AuthRateLimitResult{
			RateLimitResult: models.RateLimitResult{
				Allowed:    false,
				ResetAt:    resetAt,
				RetryAfter: retryAfter,
			},
			RequiresCaptcha: record.RequiresCaptcha,
			FailureCount:    record.FailureCount,
		}, nil
	}

	// Apply progressive backoff (FR-2b: "250ms → 500ms → 1s")
	// Calculate backoff even for zero failures to maintain constant-time behavior
	delay := s.GetProgressiveBackoff(record.FailureCount)
	remaining := min(record.RemainingAttempts(s.config.AttemptsPerWindow), s.config.AttemptsPerWindow)

	return &models.AuthRateLimitResult{
		RateLimitResult: models.RateLimitResult{
			Allowed:    true,
			Limit:      s.config.AttemptsPerWindow,
			Remaining:  remaining,
			ResetAt:    now.Add(s.config.WindowDuration),
			RetryAfter: int(delay.Milliseconds()),
		},
		RequiresCaptcha: record.RequiresCaptcha,
		FailureCount:    record.FailureCount,
	}, nil
}

func (s *Service) RecordFailure(ctx context.Context, identifier, ip string) (*models.AuthLockout, error) {
	key := models.NewAuthLockoutKey(identifier, ip).String()
	current, err := s.store.RecordFailure(ctx, key)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to record auth failure")
	}

	now := requesttime.Now(ctx)

	// Compute state transitions upfront for clarity
	shouldHardLock := current.ShouldHardLock(s.config.HardLockThreshold)
	shouldRequireCaptcha := current.ShouldRequireCaptcha(s.config.CaptchaAfterLockouts) && !current.RequiresCaptcha

	if shouldHardLock {
		current.ApplyHardLock(s.config.HardLockDuration, now)
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "auth_lockout_triggered",
			"identifier", identifier,
			"ip", privacy.AnonymizeIP(ip),
			"locked_until", current.LockedUntil,
		)
	}

	if shouldRequireCaptcha {
		current.MarkRequiresCaptcha()
	}

	if shouldHardLock || shouldRequireCaptcha {
		if err = s.store.Update(ctx, current); err != nil {
			return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to update auth lockout record")
		}
	}

	return current, nil
}

func (s *Service) Clear(ctx context.Context, identifier, ip string) error {
	key := models.NewAuthLockoutKey(identifier, ip).String()
	err := s.store.Clear(ctx, key)
	if err != nil {
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to clear auth failures")
	}

	ports.LogAudit(ctx, s.logger, s.auditPublisher, "auth_lockout_cleared",
		"identifier", identifier,
		"ip", privacy.AnonymizeIP(ip),
	)

	return nil
}

func (s *Service) GetProgressiveBackoff(failureCount int) time.Duration {
	return s.config.CalculateBackoff(failureCount)
}
