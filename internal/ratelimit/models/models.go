package models

import (
	"time"

	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"

	"github.com/google/uuid"
)

// EndpointClass categorizes endpoints for differentiated rate limiting.
type EndpointClass string

const (
	// ClassAuth: Authentication endpoints (10 req/min) - /auth/authorize, /auth/token
	ClassAuth EndpointClass = "auth"
	// ClassSensitive: Sensitive operations (30 req/min) - /consent, /vc/issue, /decision/evaluate
	ClassSensitive EndpointClass = "sensitive"
	// ClassRead: Read operations (100 req/min) - /auth/userinfo, /consent, /me/data-export
	ClassRead EndpointClass = "read"
	// ClassWrite: Write operations (50 req/min) - general mutations
	ClassWrite EndpointClass = "write"
)

// IsValid checks if the endpoint class is one of the supported enum values.
func (c EndpointClass) IsValid() bool {
	switch c {
	case ClassAuth, ClassSensitive, ClassRead, ClassWrite:
		return true
	}
	return false
}

// AllowlistEntryType defines whether an allowlist entry is for an IP or user.
type AllowlistEntryType string

const (
	AllowlistTypeIP     AllowlistEntryType = "ip"
	AllowlistTypeUserID AllowlistEntryType = "user_id"
)

// ParseAllowlistEntryType creates an AllowlistEntryType from a string, validating it.
// Returns error if the type is empty or not one of the allowed values.
func ParseAllowlistEntryType(s string) (AllowlistEntryType, error) {
	if s == "" {
		return "", dErrors.New(dErrors.CodeInvalidInput, "allowlist entry type cannot be empty")
	}
	t := AllowlistEntryType(s)
	if !t.IsValid() {
		return "", dErrors.New(dErrors.CodeInvalidInput, "invalid allowlist entry type: must be 'ip' or 'user_id'")
	}
	return t, nil
}

// IsValid checks if the allowlist entry type is one of the supported values.
func (t AllowlistEntryType) IsValid() bool {
	return t == AllowlistTypeIP || t == AllowlistTypeUserID
}

// String returns the string representation.
func (t AllowlistEntryType) String() string {
	return string(t)
}

// RateLimitResult represents the outcome of a rate limit check.
type RateLimitResult struct {
	Allowed    bool      `json:"allowed"`
	Limit      int       `json:"limit"`
	Remaining  int       `json:"remaining"`
	ResetAt    time.Time `json:"reset_at"`
	RetryAfter int       `json:"retry_after,omitempty"` // seconds, only set when not allowed
}

// AllowlistEntry represents an IP or user that bypasses rate limits.
type AllowlistEntry struct {
	ID         string             `json:"id"`
	Type       AllowlistEntryType `json:"type"`
	Identifier string             `json:"identifier"` // IP address or user_id
	Reason     string             `json:"reason"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	CreatedBy  id.UserID          `json:"created_by"` // admin user_id
}

// RateLimitViolation represents a recorded rate limit violation for audit.
type RateLimitViolation struct {
	ID            string        `json:"id"`
	Identifier    string        `json:"identifier"` // IP or user_id
	EndpointClass EndpointClass `json:"endpoint_class"`
	Endpoint      string        `json:"endpoint"`
	Limit         int           `json:"limit"`
	WindowSeconds int           `json:"window_seconds"`
	OccurredAt    time.Time     `json:"occurred_at"`
}

// QuotaTier represents partner API quota configuration.
type QuotaTier string

const (
	QuotaTierFree       QuotaTier = "free"
	QuotaTierStarter    QuotaTier = "starter"
	QuotaTierBusiness   QuotaTier = "business"
	QuotaTierEnterprise QuotaTier = "enterprise"
)

// IsValid checks if the quota tier is one of the supported enum values.
func (t QuotaTier) IsValid() bool {
	switch t {
	case QuotaTierFree, QuotaTierStarter, QuotaTierBusiness, QuotaTierEnterprise:
		return true
	}
	return false
}

// APIKeyQuota tracks quota usage for a partner API key.
type APIKeyQuota struct {
	APIKeyID       string    `json:"api_key_id"`
	Tier           QuotaTier `json:"tier"`
	MonthlyLimit   int       `json:"monthly_limit"`
	CurrentUsage   int       `json:"current_usage"`
	OverageAllowed bool      `json:"overage_allowed"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// AuthLockout tracks authentication-specific lockout state.
type AuthLockout struct {
	Identifier      string     `json:"identifier"`     // username/email + IP composite key
	FailureCount    int        `json:"failure_count"`  // failures in current window
	DailyFailures   int        `json:"daily_failures"` // failures today (for hard lock)
	LockedUntil     *time.Time `json:"locked_until,omitempty"`
	LastFailureAt   time.Time  `json:"last_failure_at"`
	RequiresCaptcha bool       `json:"requires_captcha"` // after 3 consecutive lockouts in 24h
}

// NewAllowlistEntry creates an AllowlistEntry with domain invariant validation.
func NewAllowlistEntry(entryType AllowlistEntryType, identifier, reason string, createdBy id.UserID, expiresAt *time.Time) (*AllowlistEntry, error) {
	if !entryType.IsValid() {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "invalid allowlist entry type")
	}
	if identifier == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "identifier cannot be empty")
	}
	if reason == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "reason cannot be empty")
	}

	return &AllowlistEntry{
		ID:         uuid.NewString(),
		Type:       entryType,
		Identifier: identifier,
		Reason:     reason,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
		CreatedBy:  createdBy,
	}, nil
}

// IsExpired checks if the allowlist entry has expired.
func (e *AllowlistEntry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// NewAuthLockout creates an AuthLockout with domain invariant validation.
func NewAuthLockout(identifier string) (*AuthLockout, error) {
	if identifier == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "identifier cannot be empty")
	}
	return &AuthLockout{
		Identifier:      identifier,
		FailureCount:    0,
		DailyFailures:   0,
		LockedUntil:     nil,
		LastFailureAt:   time.Now(),
		RequiresCaptcha: false,
	}, nil
}

// IsLocked checks if the account is currently locked.
func (l *AuthLockout) IsLocked() bool {
	if l.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*l.LockedUntil)
}

// NewAPIKeyQuota creates an APIKeyQuota with domain invariant validation.
func NewAPIKeyQuota(apiKeyID string, tier QuotaTier, monthlyLimit int, overageAllowed bool) (*APIKeyQuota, error) {
	if apiKeyID == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "api_key_id cannot be empty")
	}
	if !tier.IsValid() {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "invalid quota tier")
	}
	if monthlyLimit < 0 {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "monthly_limit cannot be negative")
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return &APIKeyQuota{
		APIKeyID:       apiKeyID,
		Tier:           tier,
		MonthlyLimit:   monthlyLimit,
		CurrentUsage:   0,
		OverageAllowed: overageAllowed,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}, nil
}

// IsOverQuota checks if the API key has exceeded its monthly quota.
func (q *APIKeyQuota) IsOverQuota() bool {
	return q.CurrentUsage >= q.MonthlyLimit
}

// NewRateLimitViolation creates a RateLimitViolation with domain invariant validation.
func NewRateLimitViolation(identifier string, class EndpointClass, endpoint string, limit, windowSeconds int) (*RateLimitViolation, error) {
	if identifier == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "identifier cannot be empty")
	}
	if !class.IsValid() {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "invalid endpoint class")
	}
	if endpoint == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "endpoint cannot be empty")
	}
	if limit <= 0 {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "limit must be positive")
	}
	if windowSeconds <= 0 {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "window_seconds must be positive")
	}

	return &RateLimitViolation{
		ID:            uuid.NewString(),
		Identifier:    identifier,
		EndpointClass: class,
		Endpoint:      endpoint,
		Limit:         limit,
		WindowSeconds: windowSeconds,
		OccurredAt:    time.Now(),
	}, nil
}
