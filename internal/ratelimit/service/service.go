package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"credo/internal/ratelimit/models"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/audit"
	request "credo/pkg/platform/middleware/request"
	"credo/pkg/platform/privacy"
)

type Service struct {
	buckets        BucketStore
	allowlist      AllowlistStore
	authLockout    AuthLockoutStore
	quotas         QuotaStore
	globalThrottle GlobalThrottleStore
	auditPublisher AuditPublisher
	logger         *slog.Logger
	config         *Config
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

func WithConfig(cfg *Config) Option {
	return func(s *Service) {
		s.config = cfg
	}
}

func WithAuthLockoutStore(store AuthLockoutStore) Option {
	return func(s *Service) {
		s.authLockout = store
	}
}

func WithQuotaStore(store QuotaStore) Option {
	return func(s *Service) {
		s.quotas = store
	}
}

func WithGlobalThrottleStore(store GlobalThrottleStore) Option {
	return func(s *Service) {
		s.globalThrottle = store
	}
}

const keyPrefixUser = "user"
const keyPrefixIP = "ip"
const keyPrefixAuth = "auth"

func New(
	buckets BucketStore,
	allowlist AllowlistStore,
	authLockout AuthLockoutStore,
	opts ...Option,
) (*Service, error) {
	if buckets == nil {
		return nil, fmt.Errorf("buckets store is required")
	}
	if allowlist == nil {
		return nil, fmt.Errorf("allowlist store is required")
	}
	if authLockout == nil {
		return nil, fmt.Errorf("auth lockout store is required")
	}

	svc := &Service{
		buckets:     buckets,
		allowlist:   allowlist,
		authLockout: authLockout,
		config:      DefaultConfig(),
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (s *Service) CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error) {
	requestsPerWindow, window := s.config.GetIPLimit(class)
	return s.checkRateLimit(ctx, ip, class, keyPrefixIP, requestsPerWindow, window, privacy.AnonymizeIP(ip))
}

func (s *Service) CheckUserRateLimit(ctx context.Context, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	requestsPerWindow, window := s.config.GetUserLimit(class)
	return s.checkRateLimit(ctx, userID, class, keyPrefixUser, requestsPerWindow, window, userID)
}

// checkRateLimit is the common rate limiting logic for both IP and user checks.
func (s *Service) checkRateLimit(
	ctx context.Context,
	identifier string,
	class models.EndpointClass,
	keyPrefix string,
	requestsPerWindow int,
	window time.Duration,
	logIdentifier string,
) (*models.RateLimitResult, error) {
	a, err := s.allowlist.IsAllowlisted(ctx, identifier)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check allowlist")
	}
	if a {
		return &models.RateLimitResult{
			Allowed:    true,
			Limit:      requestsPerWindow,
			Remaining:  requestsPerWindow,
			ResetAt:    time.Now().Add(window),
			RetryAfter: 0,
		}, nil
	}

	key := fmt.Sprintf("%s:%s:%s", keyPrefix, identifier, class)
	result, err := s.buckets.Allow(ctx, key, requestsPerWindow, window)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to check rate limit")
	}

	if !result.Allowed {
		s.logAudit(ctx, keyPrefix+"_rate_limit_exceeded",
			"identifier", logIdentifier,
			"endpoint_class", class,
			"limit", requestsPerWindow,
			"window_seconds", int(window.Seconds()),
		)
	}

	return result, nil
}

func (s *Service) CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error) {
	requestsPerWindow, window := s.config.GetIPLimit(class)
	ipRes, err := s.checkRateLimit(ctx, ip, class, keyPrefixIP, requestsPerWindow, window, privacy.AnonymizeIP(ip))
	if err != nil {
		return nil, err
	}
	if !ipRes.Allowed {
		return ipRes, nil
	}
	requestsPerWindow, window = s.config.GetUserLimit(class)
	userRes, err := s.checkRateLimit(ctx, userID, class, keyPrefixUser, requestsPerWindow, window, userID)
	if err != nil {
		return nil, err
	}
	if !userRes.Allowed {
		return userRes, nil
	}

	// **If both pass**, return a combined result that shows the more restrictive remaining count and reset time
	if ipRes.Remaining < userRes.Remaining {
		return ipRes, nil
	} else if userRes.Remaining < ipRes.Remaining {
		return userRes, nil
	} else {
		// If remaining counts are equal, return the one with the earlier reset time
		if ipRes.ResetAt.Before(userRes.ResetAt) {
			return ipRes, nil
		} else {
			return userRes, nil
		}
	}
}

// }- Build a key from identifier (email) and IP
// - Look up any existing lockout record
// - If no record exists, allow the request
// - If a record exists and is currently locked (LockedUntil is in the future), reject with retry-after information
// - If approaching the limit, check if should trigger a lock

func (s *Service) CheckAuthRateLimit(ctx context.Context, identifier, ip string) (*models.RateLimitResult, error) {
	// ============================================================================
	// STEP 1: Build composite key (FR-2b: "Username/email and IP combined key")
	// ============================================================================
	// The key must combine identifier (email/username) + IP to prevent:
	// - Same IP trying multiple accounts
	// - Same account being tried from multiple IPs
	//
	// TODO: Uncomment and use this key for lookups:
	// key := fmt.Sprintf("%s:%s:%s", keyPrefixAuth, identifier, ip)

	// ============================================================================
	// STEP 2: Get lockout record and handle time window expiry
	// ============================================================================
	// Get the lockout record for the composite key (not just identifier).
	// The store should handle resetting counters when windows expire:
	// - FailureCount resets after WindowDuration (15 min)
	// - DailyFailures resets after 24 hours
	//
	// TODO: Change to use composite key:
	// failureRecord, err := s.authLockout.Get(ctx, key)
	failureRecord, err := s.authLockout.Get(ctx, identifier)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to get auth lockout record")
	}

	// ============================================================================
	// STEP 3: Check if currently hard-locked (FR-2b: "hard lock for 15 minutes")
	// ============================================================================
	// If LockedUntil is set and in the future, reject immediately.
	// Include RequiresCaptcha in response if set on the record.
	if failureRecord != nil && failureRecord.LockedUntil != nil && time.Now().Before(*failureRecord.LockedUntil) {
		retryAfter := int(failureRecord.LockedUntil.Sub(time.Now()).Seconds())
		s.logAudit(ctx, "auth_lockout_triggered",
			"identifier", identifier,
			"ip", ip,
			"locked_until", failureRecord.LockedUntil,
		)
		// TODO: Return an auth-specific result type that includes RequiresCaptcha:
		// return &models.AuthRateLimitResult{
		//     Allowed:         false,
		//     RetryAfter:      retryAfter,
		//     ResetAt:         *failureRecord.LockedUntil,
		//     RequiresCaptcha: failureRecord.RequiresCaptcha,
		// }, nil
		return &models.RateLimitResult{
			Allowed:    false,
			Limit:      0,
			Remaining:  0,
			ResetAt:    *failureRecord.LockedUntil,
			RetryAfter: retryAfter,
		}, nil
	}

	// ============================================================================
	// STEP 4: Check failure count against sliding window (FR-2b: "5 attempts/15 min")
	// ============================================================================
	// Even if not hard-locked, check if FailureCount >= AttemptsPerWindow (5).
	// If at or approaching limit, either:
	// - Block the request (return Allowed: false)
	// - Or trigger a lock if threshold exceeded
	//
	// TODO: Add this check:
	// if failureRecord != nil && failureRecord.FailureCount >= s.config.AuthLockout.AttemptsPerWindow {
	//     remaining := s.config.AuthLockout.AttemptsPerWindow - failureRecord.FailureCount
	//     if remaining <= 0 {
	//         // Block - too many attempts in window
	//         resetAt := failureRecord.LastFailureAt.Add(s.config.AuthLockout.WindowDuration)
	//         return &models.RateLimitResult{
	//             Allowed:    false,
	//             Limit:      s.config.AuthLockout.AttemptsPerWindow,
	//             Remaining:  0,
	//             ResetAt:    resetAt,
	//             RetryAfter: int(resetAt.Sub(time.Now()).Seconds()),
	//         }, nil
	//     }
	// }

	// ============================================================================
	// STEP 5: Apply progressive backoff (FR-2b: "250ms → 500ms → 1s")
	// ============================================================================
	// Before returning success, apply a delay based on failure count.
	// This slows down brute-force attempts even before hard lock triggers.
	//
	// TODO: Add backoff delay:
	// if failureRecord != nil && failureRecord.FailureCount > 0 {
	//     delay := s.GetProgressiveBackoff(failureRecord.FailureCount)
	//     time.Sleep(delay) // Or return delay to caller for async handling
	// }

	// ============================================================================
	// STEP 6: Check standard IP rate limit as secondary defense
	// ============================================================================
	// This catches cases where an attacker uses many identifiers from one IP.
	// Uses the generic IP rate limit for auth class (10 req/min per FR-1).
	requestsPerWindow, window := s.config.GetIPLimit(models.ClassAuth)
	return s.checkRateLimit(ctx, ip, models.ClassAuth, keyPrefixIP, requestsPerWindow, window, privacy.AnonymizeIP(ip))

	// ============================================================================
	// STEP 7: Return combined result with lockout state
	// ============================================================================
	// The final result should include:
	// - Allowed: true/false
	// - Remaining: how many attempts left in window
	// - ResetAt: when the window resets
	// - RequiresCaptcha: if CAPTCHA verification needed (3+ lockouts in 24h)
	//
	// TODO: Create auth-specific return type in models/models.go:
	// type AuthRateLimitResult struct {
	//     RateLimitResult
	//     RequiresCaptcha bool `json:"requires_captcha"`
	//     FailureCount    int  `json:"failure_count"`
	// }
}

// RecordAuthFailure records a failed authentication attempt.
func (s *Service) RecordAuthFailure(ctx context.Context, identifier, ip string) (*models.AuthLockout, error) {
	// ============================================================================
	// BUG: Should use composite key (identifier:ip) per FR-2b
	// ============================================================================
	// TODO: Build composite key same as CheckAuthRateLimit:
	// key := fmt.Sprintf("%s:%s:%s", keyPrefixAuth, identifier, ip)
	// current, err := s.authLockout.RecordFailure(ctx, key)
	current, err := s.authLockout.RecordFailure(ctx, identifier)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to record auth failure")
	}

	if current.FailureCount >= s.config.AuthLockout.HardLockThreshold {
		lockDuration := s.config.AuthLockout.HardLockDuration
		lockedUntil := time.Now().Add(lockDuration)
		current.LockedUntil = &lockedUntil

		// ============================================================================
		// BUG: Lock is set on local struct but NEVER PERSISTED to store!
		// ============================================================================
		// The RecordFailure call above returns the updated record, but setting
		// LockedUntil here only modifies the local copy. The store never sees it.
		//
		// TODO: Either:
		// 1. Add an Update method to AuthLockoutStore interface and call it:
		//    if err := s.authLockout.Update(ctx, current); err != nil { ... }
		//
		// 2. Or have RecordFailure handle lock triggering internally based on
		//    config thresholds passed as parameters.

		//TODO: use event constant
		s.logAudit(ctx, "auth_lockout_triggered",
			"identifier", identifier,
			"ip", ip,
			"locked_until", current.LockedUntil,
		)
	}

	// Require CAPTCHA or out-of-band verification after 3 consecutive lockouts within 24 hours.
	if current.DailyFailures >= s.config.AuthLockout.CaptchaAfterLockouts {
		current.RequiresCaptcha = true
		// ============================================================================
		// BUG: RequiresCaptcha also not persisted (same issue as LockedUntil)
		// ============================================================================
	}

	// ============================================================================
	// TODO: Persist the updated record with LockedUntil and RequiresCaptcha:
	// if err := s.authLockout.Update(ctx, current); err != nil {
	//     return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to update auth lockout")
	// }
	// ============================================================================

	return current, nil
}

// ClearAuthFailures clears auth failure state after successful login.
func (s *Service) ClearAuthFailures(ctx context.Context, identifier, ip string) error {
	// TODO: Use composite key to match CheckAuthRateLimit and RecordAuthFailure:
	// key := fmt.Sprintf("%s:%s:%s", keyPrefixAuth, identifier, ip)
	// err := s.authLockout.Clear(ctx, key)
	err := s.authLockout.Clear(ctx, identifier)
	if err != nil {
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to clear auth failures")
	}

	s.logAudit(ctx, "auth_lockout_cleared",
		"identifier", identifier,
		"ip", ip,
	)

	return nil
}

// Per PRD-017 FR-2b: 250ms → 500ms → 1s (capped).
func (s *Service) GetProgressiveBackoff(failureCount int) time.Duration {
	if failureCount <= 0 {
		return 0
	}
	base := 250 * time.Millisecond
	delay := min(
		// 250ms, 500ms, 1s, 2s...
		base*time.Duration(1<<(failureCount-1)), time.Second)
	return delay
}

// CheckAPIKeyQuota checks quota for partner API key.
//
// TODO: Implement this method
// 1. Get quota for API key
// 2. Check if under monthly limit
// 3. If over limit and overage not allowed, return 429
// 4. If over limit and overage allowed, record overage
// 5. Increment usage counter
// 6. Return quota info for headers
func (s *Service) CheckAPIKeyQuota(ctx context.Context, apiKeyID string) (*models.APIKeyQuota, error) {
	// TODO: Implement - see steps above
	return nil, dErrors.New(dErrors.CodeInternal, "not implemented")
}

// CheckGlobalThrottle checks global request throttle for DDoS protection.
//
// TODO: Implement this method
// 1. Increment global counter
// 2. Check if exceeds per-instance limit
// 3. Check if exceeds global limit (Redis-backed for distributed)
// 4. If exceeded, return 503 response info
func (s *Service) CheckGlobalThrottle(ctx context.Context) (bool, error) {
	// TODO: Implement - see steps above
	return true, nil // Allow by default until implemented
}

// AddToAllowlist adds an IP or user to the rate limit allowlist.
//
// TODO: Implement this method
// 1. Validate request
// 2. Create AllowlistEntry domain object
// 3. Save to allowlist store
// 4. Emit audit event "rate_limit_allowlist_added"
func (s *Service) AddToAllowlist(ctx context.Context, req *models.AddAllowlistRequest, adminUserID string) (*models.AllowlistEntry, error) {
	// TODO: Implement - see steps above
	return nil, dErrors.New(dErrors.CodeInternal, "not implemented")
}

// RemoveFromAllowlist removes an IP or user from the allowlist.
//
// TODO: Implement this method
// 1. Validate request
// 2. Remove from allowlist store
// 3. Emit audit event "rate_limit_allowlist_removed"
func (s *Service) RemoveFromAllowlist(ctx context.Context, req *models.RemoveAllowlistRequest) error {
	// TODO: Implement
	return dErrors.New(dErrors.CodeInternal, "not implemented")
}

// ListAllowlist returns all active allowlist entries.
//
// TODO: Implement this method
func (s *Service) ListAllowlist(ctx context.Context) ([]*models.AllowlistEntry, error) {
	// TODO: Implement
	return nil, dErrors.New(dErrors.CodeInternal, "not implemented")
}

// ResetRateLimit resets the rate limit counter for an identifier.
//
// TODO: Implement this method
// 1. Validate request
// 2. Build key(s) to reset
// 3. Call buckets.Reset() for each key
// 4. Emit audit event "rate_limit_reset"
func (s *Service) ResetRateLimit(ctx context.Context, req *models.ResetRateLimitRequest) error {
	// TODO: Implement
	return dErrors.New(dErrors.CodeInternal, "not implemented")
}

// logAudit emits an audit event for rate limiting operations.
func (s *Service) logAudit(ctx context.Context, event string, attrs ...any) {
	if requestID := request.GetRequestID(ctx); requestID != "" {
		attrs = append(attrs, "request_id", requestID)
	}
	args := append(attrs, "event", event, "log_type", "audit")
	if s.logger != nil {
		s.logger.InfoContext(ctx, event, args...)
	}
	if s.auditPublisher == nil {
		return
	}
	// TODO: Extract user_id from attrs and emit audit event
	_ = s.auditPublisher.Emit(ctx, audit.Event{
		Action: event,
	})
}
