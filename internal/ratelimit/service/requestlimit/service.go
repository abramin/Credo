package requestlimit

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"credo/internal/ratelimit/config"
	"credo/internal/ratelimit/metrics"
	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/ports"
	dErrors "credo/pkg/domain-errors"
	requesttime "credo/pkg/platform/middleware/requesttime"
	"credo/pkg/platform/privacy"
)

// Type aliases for interfaces from ports package.
// This allows external packages to use these types without importing ports directly.
type (
	BucketStore    = ports.BucketStore
	AllowlistStore = ports.AllowlistStore
	AuditPublisher = ports.AuditPublisher
)

type Service struct {
	buckets        BucketStore
	allowlist      AllowlistStore
	auditPublisher AuditPublisher
	logger         *slog.Logger
	config         *config.Config
	metrics        *metrics.Metrics
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

func WithConfig(cfg *config.Config) Option {
	return func(s *Service) {
		s.config = cfg
	}
}

func WithMetrics(m *metrics.Metrics) Option {
	return func(s *Service) {
		s.metrics = m
	}
}

func New(
	buckets BucketStore,
	allowlist AllowlistStore,
	opts ...Option,
) (*Service, error) {
	if buckets == nil {
		return nil, errors.New("buckets store is required")
	}
	if allowlist == nil {
		return nil, errors.New("allowlist store is required")
	}

	svc := &Service{
		buckets:   buckets,
		allowlist: allowlist,
		config:    config.DefaultConfig(),
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (s *Service) CheckIP(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error) {
	requestsPerWindow, window, ok := s.config.GetIPLimit(class)
	if !ok {
		// Default-deny: no limit configured for this class (PRD-017 FR-1)
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "rate_limit_config_missing",
			"identifier", privacy.AnonymizeIP(ip),
			"endpoint_class", class,
			"limit_type", models.KeyPrefixIP,
		)
		return &models.RateLimitResult{
			Allowed:    false,
			Limit:      0,
			Remaining:  0,
			ResetAt:    requesttime.Now(ctx),
			RetryAfter: 60, // Retry in 60 seconds
		}, nil
	}
	return s.checkRateLimit(ctx, ip, class, models.KeyPrefixIP, requestsPerWindow, window, privacy.AnonymizeIP(ip))
}

func (s *Service) CheckUser(ctx context.Context, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	requestsPerWindow, window, ok := s.config.GetUserLimit(class)
	if !ok {
		// Default-deny: no limit configured for this class (PRD-017 FR-1)
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "rate_limit_config_missing",
			"identifier", userID,
			"endpoint_class", class,
			"limit_type", models.KeyPrefixUser,
		)
		return &models.RateLimitResult{
			Allowed:    false,
			Limit:      0,
			Remaining:  0,
			ResetAt:    requesttime.Now(ctx),
			RetryAfter: 60, // Retry in 60 seconds
		}, nil
	}
	return s.checkRateLimit(ctx, userID, class, models.KeyPrefixUser, requestsPerWindow, window, userID)
}

func (s *Service) checkRateLimit(
	ctx context.Context,
	identifier string,
	class models.EndpointClass,
	keyPrefix models.KeyPrefix,
	requestsPerWindow int,
	window time.Duration,
	logIdentifier string,
) (*models.RateLimitResult, error) {
	now := requesttime.Now(ctx)

	a, err := s.allowlist.IsAllowlisted(ctx, identifier)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check allowlist")
	}
	if a {
		// Record allowlist bypass metrics and audit
		bypassType := string(keyPrefix)
		if s.metrics != nil {
			s.metrics.RecordAllowlistBypass(bypassType)
		}
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "allowlist_bypass",
			"identifier", logIdentifier,
			"endpoint_class", class,
			"bypass_type", bypassType,
		)
		return &models.RateLimitResult{
			Allowed:    true,
			Bypassed:   true,
			Limit:      requestsPerWindow,
			Remaining:  requestsPerWindow,
			ResetAt:    now.Add(window),
			RetryAfter: 0,
		}, nil
	}

	key := models.NewRateLimitKey(keyPrefix, identifier, class)
	result, err := s.buckets.Allow(ctx, key.String(), requestsPerWindow, window)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check rate limit")
	}

	if !result.Allowed {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, string(keyPrefix)+"_rate_limit_exceeded",
			"identifier", logIdentifier,
			"endpoint_class", class,
			"limit", requestsPerWindow,
			"window_seconds", int(window.Seconds()),
		)
	}

	return result, nil
}

func (s *Service) CheckBoth(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	now := requesttime.Now(ctx)

	// Get limits upfront to fail fast if config is missing
	ipRequestsPerWindow, ipWindow, ipOk := s.config.GetIPLimit(class)
	if !ipOk {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "rate_limit_config_missing",
			"identifier", privacy.AnonymizeIP(ip),
			"endpoint_class", class,
			"limit_type", models.KeyPrefixIP,
		)
		return &models.RateLimitResult{
			Allowed:    false,
			Limit:      0,
			Remaining:  0,
			ResetAt:    now,
			RetryAfter: 60,
		}, nil
	}

	userRequestsPerWindow, userWindow, userOk := s.config.GetUserLimit(class)
	if !userOk {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "rate_limit_config_missing",
			"identifier", userID,
			"endpoint_class", class,
			"limit_type", models.KeyPrefixUser,
		)
		return &models.RateLimitResult{
			Allowed:    false,
			Limit:      0,
			Remaining:  0,
			ResetAt:    now,
			RetryAfter: 60,
		}, nil
	}

	// Single allowlist check for both identifiers upfront (optimization: avoid duplicate checks)
	ipAllowlisted, err := s.allowlist.IsAllowlisted(ctx, ip)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check IP allowlist")
	}
	userAllowlisted, err := s.allowlist.IsAllowlisted(ctx, userID)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check user allowlist")
	}

	// If either is allowlisted, bypass rate limiting entirely
	if ipAllowlisted || userAllowlisted {
		bypassType := "ip"
		if userAllowlisted {
			bypassType = "user"
		}
		if s.metrics != nil {
			s.metrics.RecordAllowlistBypass(bypassType)
		}
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "allowlist_bypass",
			"ip", privacy.AnonymizeIP(ip),
			"user_id", userID,
			"endpoint_class", class,
			"bypass_type", bypassType,
		)
		// Return the more restrictive limit info for consistency
		limit, window := ipRequestsPerWindow, ipWindow
		if userRequestsPerWindow < ipRequestsPerWindow {
			limit, window = userRequestsPerWindow, userWindow
		}
		return &models.RateLimitResult{
			Allowed:    true,
			Bypassed:   true,
			Limit:      limit,
			Remaining:  limit,
			ResetAt:    now.Add(window),
			RetryAfter: 0,
		}, nil
	}

	// Check IP rate limit
	ipKey := models.NewRateLimitKey(models.KeyPrefixIP, ip, class)
	ipRes, err := s.buckets.Allow(ctx, ipKey.String(), ipRequestsPerWindow, ipWindow)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check IP rate limit")
	}
	if !ipRes.Allowed {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "ip_rate_limit_exceeded",
			"identifier", privacy.AnonymizeIP(ip),
			"endpoint_class", class,
			"limit", ipRequestsPerWindow,
			"window_seconds", int(ipWindow.Seconds()),
		)
		return ipRes, nil
	}

	// Check user rate limit
	userKey := models.NewRateLimitKey(models.KeyPrefixUser, userID, class)
	userRes, err := s.buckets.Allow(ctx, userKey.String(), userRequestsPerWindow, userWindow)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check user rate limit")
	}
	if !userRes.Allowed {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "user_rate_limit_exceeded",
			"identifier", userID,
			"endpoint_class", class,
			"limit", userRequestsPerWindow,
			"window_seconds", int(userWindow.Seconds()),
		)
		return userRes, nil
	}

	// Return the more restrictive result
	return moreRestrictiveResult(ipRes, userRes), nil
}

// moreRestrictiveResult returns the result with fewer remaining requests,
// or the earlier reset time if remaining counts are equal.
func moreRestrictiveResult(a, b *models.RateLimitResult) *models.RateLimitResult {
	if a.Remaining < b.Remaining {
		return a
	}
	if b.Remaining < a.Remaining {
		return b
	}
	if a.ResetAt.Before(b.ResetAt) {
		return a
	}
	return b
}
