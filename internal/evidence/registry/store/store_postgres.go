package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"credo/internal/evidence/registry/metrics"
	"credo/internal/evidence/registry/models"
	registrysqlc "credo/internal/evidence/registry/store/sqlc"
	id "credo/pkg/domain"
	"credo/pkg/requestcontext"
)

// PostgresCache persists registry cache entries in PostgreSQL.
type PostgresCache struct {
	db       *sql.DB
	cacheTTL time.Duration
	metrics  *metrics.Metrics
	queries  *registrysqlc.Queries
}

// NewPostgresCache constructs a PostgreSQL-backed registry cache.
func NewPostgresCache(db *sql.DB, cacheTTL time.Duration, metrics *metrics.Metrics) *PostgresCache {
	return &PostgresCache{
		db:       db,
		cacheTTL: cacheTTL,
		metrics:  metrics,
		queries:  registrysqlc.New(db),
	}
}

func (c *PostgresCache) FindCitizen(ctx context.Context, nationalID id.NationalID, regulated bool) (*models.CitizenRecord, error) {
	start := time.Now()
	cutoff := requestcontext.Now(ctx).Add(-c.cacheTTL)
	record, err := c.queries.GetCitizenCache(ctx, registrysqlc.GetCitizenCacheParams{
		NationalID: nationalID.String(),
		Regulated:  regulated,
		CheckedAt:  cutoff,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.recordMiss("citizen", start)
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find citizen cache: %w", err)
	}
	c.recordHit("citizen", start)
	return toCitizenRecord(record), nil
}

func (c *PostgresCache) SaveCitizen(ctx context.Context, key id.NationalID, record *models.CitizenRecord, regulated bool) error {
	if record == nil {
		return fmt.Errorf("citizen record is required")
	}
	err := c.queries.UpsertCitizenCache(ctx, registrysqlc.UpsertCitizenCacheParams{
		NationalID:  key.String(),
		FullName:    record.FullName,
		DateOfBirth: record.DateOfBirth,
		Address:     record.Address,
		Valid:       record.Valid,
		Source:      record.Source,
		CheckedAt:   record.CheckedAt,
		Regulated:   regulated,
	})
	if err != nil {
		return fmt.Errorf("save citizen cache: %w", err)
	}
	return nil
}

func (c *PostgresCache) FindSanction(ctx context.Context, nationalID id.NationalID) (*models.SanctionsRecord, error) {
	start := time.Now()
	cutoff := requestcontext.Now(ctx).Add(-c.cacheTTL)
	record, err := c.queries.GetSanctionsCache(ctx, registrysqlc.GetSanctionsCacheParams{
		NationalID: nationalID.String(),
		CheckedAt:  cutoff,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.recordMiss("sanctions", start)
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find sanctions cache: %w", err)
	}
	c.recordHit("sanctions", start)
	return toSanctionsRecord(record), nil
}

func (c *PostgresCache) SaveSanction(ctx context.Context, key id.NationalID, record *models.SanctionsRecord) error {
	if record == nil {
		return fmt.Errorf("sanctions record is required")
	}
	err := c.queries.UpsertSanctionsCache(ctx, registrysqlc.UpsertSanctionsCacheParams{
		NationalID: key.String(),
		Listed:     record.Listed,
		Source:     record.Source,
		CheckedAt:  record.CheckedAt,
	})
	if err != nil {
		return fmt.Errorf("save sanctions cache: %w", err)
	}
	return nil
}

func toCitizenRecord(record registrysqlc.CitizenCache) *models.CitizenRecord {
	return &models.CitizenRecord{
		NationalID:  record.NationalID,
		FullName:    record.FullName,
		DateOfBirth: record.DateOfBirth,
		Address:     record.Address,
		Valid:       record.Valid,
		Source:      record.Source,
		CheckedAt:   record.CheckedAt,
	}
}

func toSanctionsRecord(record registrysqlc.SanctionsCache) *models.SanctionsRecord {
	return &models.SanctionsRecord{
		NationalID: record.NationalID,
		Listed:     record.Listed,
		Source:     record.Source,
		CheckedAt:  record.CheckedAt,
	}
}

func (c *PostgresCache) recordHit(recordType string, start time.Time) {
	if c.metrics == nil {
		return
	}
	c.metrics.RecordCacheHit(recordType)
	c.metrics.ObserveLookupDuration(recordType, time.Since(start).Seconds())
}

func (c *PostgresCache) recordMiss(recordType string, start time.Time) {
	if c.metrics == nil {
		return
	}
	c.metrics.RecordCacheMiss(recordType)
	c.metrics.ObserveLookupDuration(recordType, time.Since(start).Seconds())
}
