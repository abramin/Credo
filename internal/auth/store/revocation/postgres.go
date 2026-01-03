package revocation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// PostgresTRL persists revoked token JTIs in PostgreSQL.
type PostgresTRL struct {
	db    *sql.DB
	clock Clock // injected clock for testability (defaults to time.Now)
}

// PostgresTRLOption configures a PostgresTRL instance.
type PostgresTRLOption func(*PostgresTRL)

// WithPostgresClock sets the clock function for testability.
func WithPostgresClock(clock Clock) PostgresTRLOption {
	return func(trl *PostgresTRL) {
		if clock != nil {
			trl.clock = clock
		}
	}
}

// NewPostgresTRL constructs a PostgreSQL-backed token revocation list.
func NewPostgresTRL(db *sql.DB, opts ...PostgresTRLOption) *PostgresTRL {
	trl := &PostgresTRL{
		db:    db,
		clock: time.Now, // default to real time
	}
	for _, opt := range opts {
		if opt != nil {
			opt(trl)
		}
	}
	return trl
}

// RevokeToken adds a token to the revocation list with TTL.
func (t *PostgresTRL) RevokeToken(ctx context.Context, jti string, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}
	expiresAt := t.clock().Add(ttl)
	query := `
		INSERT INTO token_revocations (jti, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (jti) DO UPDATE SET
			expires_at = EXCLUDED.expires_at
	`
	_, err := t.db.ExecContext(ctx, query, jti, expiresAt)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// IsRevoked checks if a token is in the revocation list.
func (t *PostgresTRL) IsRevoked(ctx context.Context, jti string) (bool, error) {
	var expiresAt time.Time
	err := t.db.QueryRowContext(ctx, `SELECT expires_at FROM token_revocations WHERE jti = $1`, jti).Scan(&expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check token revocation: %w", err)
	}
	if t.clock().After(expiresAt) {
		return false, nil
	}
	return true, nil
}

// RevokeSessionTokens revokes multiple tokens associated with a session.
// Uses batch INSERT with unnest for efficiency instead of per-row inserts.
func (t *PostgresTRL) RevokeSessionTokens(ctx context.Context, sessionID string, jtis []string, ttl time.Duration) error {
	if len(jtis) == 0 {
		return nil
	}
	if err := validateTTL(ttl); err != nil {
		return err
	}

	// Filter empty JTIs
	validJTIs := make([]string, 0, len(jtis))
	for _, jti := range jtis {
		if jti != "" {
			validJTIs = append(validJTIs, jti)
		}
	}
	if len(validJTIs) == 0 {
		return nil
	}

	expiresAt := t.clock().Add(ttl)

	// Batch insert using unnest for O(1) round trips instead of O(n)
	query := `
		INSERT INTO token_revocations (jti, expires_at)
		SELECT unnest($1::text[]), $2
		ON CONFLICT (jti) DO UPDATE SET
			expires_at = EXCLUDED.expires_at
	`
	_, err := t.db.ExecContext(ctx, query, pq.Array(validJTIs), expiresAt)
	if err != nil {
		return fmt.Errorf("revoke session tokens batch: %w", err)
	}
	return nil
}
