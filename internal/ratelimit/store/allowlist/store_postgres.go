package allowlist

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"credo/internal/ratelimit/models"
	ratelimitsqlc "credo/internal/ratelimit/store/sqlc"
	id "credo/pkg/domain"
	"credo/pkg/requestcontext"

	"github.com/google/uuid"
)

// PostgresStore persists allowlist entries in PostgreSQL.
type PostgresStore struct {
	db      *sql.DB
	queries *ratelimitsqlc.Queries
}

// NewPostgres constructs a PostgreSQL-backed allowlist store.
func NewPostgres(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db:      db,
		queries: ratelimitsqlc.New(db),
	}
}

func (s *PostgresStore) Add(ctx context.Context, entry *models.AllowlistEntry) error {
	if entry == nil {
		return fmt.Errorf("allowlist entry is required")
	}
	err := s.queries.UpsertAllowlistEntry(ctx, ratelimitsqlc.UpsertAllowlistEntryParams{
		ID:         entry.ID,
		EntryType:  string(entry.Type),
		Identifier: entry.Identifier.String(),
		Reason:     entry.Reason,
		ExpiresAt:  nullTime(entry.ExpiresAt),
		CreatedAt:  entry.CreatedAt,
		CreatedBy:  uuid.UUID(entry.CreatedBy),
	})
	if err != nil {
		return fmt.Errorf("add allowlist entry: %w", err)
	}
	return nil
}

func (s *PostgresStore) Remove(ctx context.Context, entryType models.AllowlistEntryType, identifier string) error {
	err := s.queries.DeleteAllowlistEntry(ctx, ratelimitsqlc.DeleteAllowlistEntryParams{
		EntryType:  string(entryType),
		Identifier: identifier,
	})
	if err != nil {
		return fmt.Errorf("remove allowlist entry: %w", err)
	}
	return nil
}

func (s *PostgresStore) IsAllowlisted(ctx context.Context, identifier string) (bool, error) {
	if identifier == "" {
		return false, nil
	}
	now := requestcontext.Now(ctx)
	exists, err := s.queries.IsAllowlisted(ctx, ratelimitsqlc.IsAllowlistedParams{
		Identifier: identifier,
		ExpiresAt:  sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("check allowlist: %w", err)
	}
	return exists, nil
}

func (s *PostgresStore) List(ctx context.Context) ([]*models.AllowlistEntry, error) {
	now := requestcontext.Now(ctx)
	rows, err := s.queries.ListAllowlistEntries(ctx, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list allowlist entries: %w", err)
	}
	entries := make([]*models.AllowlistEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, toAllowlistEntry(row))
	}
	return entries, nil
}

// StartCleanup runs periodic cleanup of expired entries until ctx is cancelled.
func (s *PostgresStore) StartCleanup(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.RemoveExpiredAt(ctx, time.Now()); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// RemoveExpiredAt removes all entries that have expired as of the given time.
// Exported for testability; background cleanup passes wall-clock time.
func (s *PostgresStore) RemoveExpiredAt(ctx context.Context, now time.Time) error {
	if err := s.queries.DeleteExpiredAllowlistEntries(ctx, sql.NullTime{Time: now, Valid: true}); err != nil {
		return fmt.Errorf("cleanup allowlist entries: %w", err)
	}
	return nil
}

func toAllowlistEntry(row ratelimitsqlc.RateLimitAllowlist) *models.AllowlistEntry {
	entry := &models.AllowlistEntry{
		ID:         row.ID,
		Type:       models.AllowlistEntryType(row.EntryType),
		Identifier: models.AllowlistIdentifier(row.Identifier),
		Reason:     row.Reason,
		CreatedAt:  row.CreatedAt,
		CreatedBy:  id.UserID(row.CreatedBy),
	}
	if row.ExpiresAt.Valid {
		entry.ExpiresAt = &row.ExpiresAt.Time
	}
	return entry
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}
