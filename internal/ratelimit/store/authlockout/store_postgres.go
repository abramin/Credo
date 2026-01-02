package authlockout

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"credo/internal/ratelimit/config"
	"credo/internal/ratelimit/models"
)

// PostgresStore persists auth lockout records in PostgreSQL.
// This store is pure I/O—all domain logic (lock checks, cutoff calculations) belongs in the service.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgres constructs a PostgreSQL-backed auth lockout store.
// The config parameter is accepted for API compatibility but stores don't use config
// (business rules belong in the service layer).
func NewPostgres(db *sql.DB, _ *config.AuthLockoutConfig) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, identifier string) (*models.AuthLockout, error) {
	query := `
		SELECT identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha
		FROM auth_lockouts
		WHERE identifier = $1
	`
	record, err := scanAuthLockout(s.db.QueryRowContext(ctx, query, identifier))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get auth lockout: %w", err)
	}
	return record, nil
}

// GetOrCreate retrieves an existing lockout record or creates a new one with zero counts.
// This is pure I/O—the service owns counter increments via domain methods.
func (s *PostgresStore) GetOrCreate(ctx context.Context, identifier string, now time.Time) (*models.AuthLockout, error) {
	query := `
		INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
		VALUES ($1, 0, 0, NULL, $2, FALSE)
		ON CONFLICT (identifier) DO UPDATE SET
			identifier = EXCLUDED.identifier
		RETURNING identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha
	`
	record, err := scanAuthLockout(s.db.QueryRowContext(ctx, query, identifier, now))
	if err != nil {
		return nil, fmt.Errorf("get or create auth lockout: %w", err)
	}
	return record, nil
}

func (s *PostgresStore) Clear(ctx context.Context, identifier string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_lockouts WHERE identifier = $1`, identifier)
	if err != nil {
		return fmt.Errorf("clear auth lockout: %w", err)
	}
	return nil
}

func (s *PostgresStore) Update(ctx context.Context, record *models.AuthLockout) error {
	if record == nil {
		return fmt.Errorf("auth lockout record is required")
	}
	query := `
		INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (identifier) DO UPDATE SET
			failure_count = EXCLUDED.failure_count,
			daily_failures = EXCLUDED.daily_failures,
			locked_until = EXCLUDED.locked_until,
			last_failure_at = EXCLUDED.last_failure_at,
			requires_captcha = EXCLUDED.requires_captcha
	`
	_, err := s.db.ExecContext(ctx, query,
		record.Identifier,
		record.FailureCount,
		record.DailyFailures,
		record.LockedUntil,
		record.LastFailureAt,
		record.RequiresCaptcha,
	)
	if err != nil {
		return fmt.Errorf("update auth lockout: %w", err)
	}
	return nil
}

// RecordFailureAtomic atomically increments failure counts and returns the updated record.
// This prevents TOCTOU races where concurrent requests could bypass hard lock thresholds.
// The method uses a single atomic UPDATE...RETURNING to ensure consistency.
func (s *PostgresStore) RecordFailureAtomic(ctx context.Context, identifier string, now time.Time) (*models.AuthLockout, error) {
	query := `
		INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
		VALUES ($1, 1, 1, NULL, $2, FALSE)
		ON CONFLICT (identifier) DO UPDATE SET
			failure_count = auth_lockouts.failure_count + 1,
			daily_failures = auth_lockouts.daily_failures + 1,
			last_failure_at = $2
		RETURNING identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha
	`
	record, err := scanAuthLockout(s.db.QueryRowContext(ctx, query, identifier, now))
	if err != nil {
		return nil, fmt.Errorf("record failure atomic: %w", err)
	}
	return record, nil
}

// ApplyHardLockAtomic atomically sets the hard lock if thresholds are met.
// Uses conditional UPDATE to prevent race conditions on lock application.
func (s *PostgresStore) ApplyHardLockAtomic(ctx context.Context, identifier string, lockedUntil time.Time, dailyThreshold int) (applied bool, err error) {
	query := `
		UPDATE auth_lockouts
		SET locked_until = $2
		WHERE identifier = $1
		  AND daily_failures >= $3
		  AND (locked_until IS NULL OR locked_until < NOW())
	`
	result, err := s.db.ExecContext(ctx, query, identifier, lockedUntil, dailyThreshold)
	if err != nil {
		return false, fmt.Errorf("apply hard lock atomic: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("apply hard lock rows affected: %w", err)
	}
	return rows > 0, nil
}

// SetRequiresCaptchaAtomic atomically sets the CAPTCHA requirement if thresholds are met.
func (s *PostgresStore) SetRequiresCaptchaAtomic(ctx context.Context, identifier string, lockoutThreshold int) (applied bool, err error) {
	// Count consecutive lockouts by checking daily_failures threshold breaches
	// This is a simplified check - in practice you might track lockout_count separately
	query := `
		UPDATE auth_lockouts
		SET requires_captcha = TRUE
		WHERE identifier = $1
		  AND requires_captcha = FALSE
		  AND daily_failures >= $2
	`
	result, err := s.db.ExecContext(ctx, query, identifier, lockoutThreshold)
	if err != nil {
		return false, fmt.Errorf("set requires captcha atomic: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("set requires captcha rows affected: %w", err)
	}
	return rows > 0, nil
}

// ResetFailureCount resets window failure counts for records with last_failure_at before cutoff.
// The cutoff is provided by the caller to keep business rules (window duration) out of the store.
func (s *PostgresStore) ResetFailureCount(ctx context.Context, cutoff time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin reset failure count: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(failure_count), 0) FROM auth_lockouts WHERE last_failure_at < $1`, cutoff).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum failure count: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE auth_lockouts SET failure_count = 0 WHERE last_failure_at < $1`, cutoff); err != nil {
		return 0, fmt.Errorf("reset failure count: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit reset failure count: %w", err)
	}
	return total, nil
}

// ResetDailyFailures resets daily failure counts for records with last_failure_at before cutoff.
// The cutoff is provided by the caller to keep business rules (24h window) out of the store.
func (s *PostgresStore) ResetDailyFailures(ctx context.Context, cutoff time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin reset daily failures: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(daily_failures), 0) FROM auth_lockouts WHERE last_failure_at < $1`, cutoff).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum daily failures: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE auth_lockouts SET daily_failures = 0 WHERE last_failure_at < $1`, cutoff); err != nil {
		return 0, fmt.Errorf("reset daily failures: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit reset daily failures: %w", err)
	}
	return total, nil
}

type authLockoutRow interface {
	Scan(dest ...any) error
}

func scanAuthLockout(row authLockoutRow) (*models.AuthLockout, error) {
	var record models.AuthLockout
	var lockedUntil sql.NullTime
	if err := row.Scan(&record.Identifier, &record.FailureCount, &record.DailyFailures, &lockedUntil, &record.LastFailureAt, &record.RequiresCaptcha); err != nil {
		return nil, err
	}
	if lockedUntil.Valid {
		record.LockedUntil = &lockedUntil.Time
	}
	return &record, nil
}
