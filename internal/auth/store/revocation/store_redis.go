package revocation

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	isRevokedDurationMs = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "credo_is_token_revoked_duration_ms",
		Help:    "Latency of token revocation checks in milliseconds",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25},
	})
)

const (
	// Redis key prefix for revoked tokens
	revokedTokenKeyPrefix = "trl:jti:"
)

// RedisTRL is a Redis-backed implementation of TokenRevocationList.
// This is the production-recommended implementation for distributed deployments
// where multiple instances need to share token revocation state.
type RedisTRL struct {
	client *redis.Client
}

// RedisTRLOption configures a RedisTRL instance.
type RedisTRLOption func(*RedisTRL)

// NewRedisTRL constructs a Redis-backed token revocation list.
func NewRedisTRL(client *redis.Client, opts ...RedisTRLOption) *RedisTRL {
	trl := &RedisTRL{
		client: client,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(trl)
		}
	}
	return trl
}

// RevokeToken adds a token to the revocation list with TTL.
// Uses Redis SETEX for atomic set-with-expiry.
func (t *RedisTRL) RevokeToken(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return nil
	}
	key := revokedTokenKeyPrefix + jti
	// Store "1" as a simple marker; the key existence is what matters
	return t.client.Set(ctx, key, "1", ttl).Err()
}

// IsRevoked checks if a token is in the revocation list.
// Returns false if the key doesn't exist (not revoked or expired).
func (t *RedisTRL) IsRevoked(ctx context.Context, jti string) (bool, error) {
	start := time.Now()
	defer func() {
		isRevokedDurationMs.Observe(float64(time.Since(start).Microseconds()) / 1000.0)
	}()

	if jti == "" {
		return false, nil
	}
	key := revokedTokenKeyPrefix + jti
	_, err := t.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RevokeSessionTokens revokes multiple tokens associated with a session.
// Uses Redis pipeline for efficiency.
func (t *RedisTRL) RevokeSessionTokens(ctx context.Context, sessionID string, jtis []string, ttl time.Duration) error {
	if len(jtis) == 0 {
		return nil
	}

	// Use pipeline for batch operations
	pipe := t.client.Pipeline()
	for _, jti := range jtis {
		if jti != "" {
			key := revokedTokenKeyPrefix + jti
			pipe.Set(ctx, key, "1", ttl)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Close is a no-op for RedisTRL since the client lifecycle is managed externally.
func (t *RedisTRL) Close() {
	// Client lifecycle managed externally
}
