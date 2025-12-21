package service

import (
	"context"
	"time"

	"credo/pkg/platform/audit"
	"credo/internal/ratelimit/models"
)

// BucketStore defines the persistence interface for rate limit buckets/counters.
// Keys are simple strings - validation happens at the boundary (middleware/handler).
type BucketStore interface {
	// Allow checks if a request is allowed and increments the counter.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*models.RateLimitResult, error)

	// AllowN checks if a request with custom cost is allowed.
	AllowN(ctx context.Context, key string, cost, limit int, window time.Duration) (*models.RateLimitResult, error)

	// Reset clears the rate limit counter for a key.
	Reset(ctx context.Context, key string) error

	// GetCurrentCount returns the current request count for a key.
	// Used for monitoring and admin display.
	GetCurrentCount(ctx context.Context, key string) (int, error)
}

// AllowlistStore defines the persistence interface for rate limit allowlist.
type AllowlistStore interface {
	// Add adds an identifier to the allowlist.
	Add(ctx context.Context, entry *models.AllowlistEntry) error

	// Remove removes an identifier from the allowlist.
	Remove(ctx context.Context, entryType models.AllowlistEntryType, identifier string) error

	// IsAllowlisted checks if an identifier is in the allowlist and not expired.
	IsAllowlisted(ctx context.Context, identifier string) (bool, error)

	// List returns all active allowlist entries.
	List(ctx context.Context) ([]*models.AllowlistEntry, error)
}

// AuthLockoutStore defines the persistence interface for authentication lockouts.
type AuthLockoutStore interface {
	// RecordFailure records a failed authentication attempt.
	RecordFailure(ctx context.Context, identifier string) (*models.AuthLockout, error)

	// GetLockout retrieves the current lockout state for an identifier.
	GetLockout(ctx context.Context, identifier string) (*models.AuthLockout, error)

	// ClearLockout clears the lockout state after successful authentication.
	ClearLockout(ctx context.Context, identifier string) error

	// IsLocked checks if an identifier is currently locked out.
	IsLocked(ctx context.Context, identifier string) (bool, *time.Time, error)
}

// QuotaStore defines the persistence interface for partner API quotas.
type QuotaStore interface {
	// GetQuota retrieves the quota for an API key.
	GetQuota(ctx context.Context, apiKeyID string) (*models.APIKeyQuota, error)

	// IncrementUsage increments the usage counter for an API key.
	IncrementUsage(ctx context.Context, apiKeyID string, count int) (*models.APIKeyQuota, error)

	// SetQuota sets or updates the quota configuration for an API key.
	SetQuota(ctx context.Context, quota *models.APIKeyQuota) error
}

// AuditPublisher defines the interface for publishing audit events.
type AuditPublisher interface {
	Emit(ctx context.Context, event audit.Event) error
}

// GlobalThrottleStore defines the interface for global request throttling.
type GlobalThrottleStore interface {
	// IncrementGlobal increments the global request counter.
	// Returns current count and whether limit is exceeded.
	IncrementGlobal(ctx context.Context) (int, bool, error)

	// GetGlobalCount returns the current global request count.
	GetGlobalCount(ctx context.Context) (int, error)
}
