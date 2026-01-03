package globalthrottle

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"credo/internal/ratelimit/config"
	ratelimitsqlc "credo/internal/ratelimit/store/sqlc"
	"credo/pkg/requestcontext"
)

const (
	bucketSecond = "second"
	bucketHour   = "hour"
)

// PostgresStore persists global throttle counters in PostgreSQL.
type PostgresStore struct {
	db             *sql.DB
	perSecondLimit int
	perHourLimit   int
	queries        *ratelimitsqlc.Queries
}

// NewPostgres constructs a PostgreSQL-backed global throttle store.
func NewPostgres(db *sql.DB, cfg *config.GlobalLimit) *PostgresStore {
	if cfg == nil {
		defaultCfg := config.DefaultConfig().Global
		cfg = &defaultCfg
	}
	return &PostgresStore{
		db:             db,
		perSecondLimit: cfg.GlobalPerSecond,
		perHourLimit:   cfg.PerInstancePerHour,
		queries:        ratelimitsqlc.New(db),
	}
}

// IncrementGlobal increments the global counter and checks if the request is blocked.
func (s *PostgresStore) IncrementGlobal(ctx context.Context) (count int, blocked bool, err error) {
	now := requestcontext.Now(ctx)
	currentSecond := now.Truncate(time.Second)
	currentHour := now.Truncate(time.Hour)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, fmt.Errorf("begin global throttle tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.queries.WithTx(tx)
	secStart, secCount, err := s.loadBucket(ctx, qtx, bucketSecond, currentSecond)
	if err != nil {
		return 0, false, err
	}
	hourStart, hourCount, err := s.loadBucket(ctx, qtx, bucketHour, currentHour)
	if err != nil {
		return 0, false, err
	}

	if secCount+1 > s.perSecondLimit {
		if err := tx.Commit(); err != nil {
			return 0, false, fmt.Errorf("commit global throttle tx: %w", err)
		}
		return secCount, true, nil
	}
	if hourCount+1 > s.perHourLimit {
		if err := tx.Commit(); err != nil {
			return 0, false, fmt.Errorf("commit global throttle tx: %w", err)
		}
		return hourCount, true, nil
	}

	secCount++
	hourCount++
	if err := s.updateBucket(ctx, qtx, bucketSecond, secStart, secCount); err != nil {
		return 0, false, err
	}
	if err := s.updateBucket(ctx, qtx, bucketHour, hourStart, hourCount); err != nil {
		return 0, false, err
	}

	if err := tx.Commit(); err != nil {
		return 0, false, fmt.Errorf("commit global throttle tx: %w", err)
	}
	return secCount, false, nil
}

// GetGlobalCount returns the current count in the per-second window.
func (s *PostgresStore) GetGlobalCount(ctx context.Context) (count int, err error) {
	now := requestcontext.Now(ctx).Truncate(time.Second)
	var bucketStart time.Time
	var current int32
	row, err := s.queries.GetGlobalThrottleBucket(ctx, bucketSecond)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("get global count: %w", err)
	}
	bucketStart = row.BucketStart
	current = row.Count
	if bucketStart != now {
		return 0, nil
	}
	return int(current), nil
}

func (s *PostgresStore) loadBucket(ctx context.Context, queries *ratelimitsqlc.Queries, bucketType string, current time.Time) (time.Time, int, error) {
	var bucketStart time.Time
	var count int32
	row, err := queries.GetGlobalThrottleBucketForUpdate(ctx, bucketType)
	if err != nil {
		if err == sql.ErrNoRows {
			if err := queries.InsertGlobalThrottleBucket(ctx, ratelimitsqlc.InsertGlobalThrottleBucketParams{
				BucketType:  bucketType,
				BucketStart: current,
			}); err != nil {
				return time.Time{}, 0, fmt.Errorf("insert global throttle bucket: %w", err)
			}
			row, err = queries.GetGlobalThrottleBucketForUpdate(ctx, bucketType)
			if err != nil {
				return time.Time{}, 0, fmt.Errorf("reload global throttle bucket: %w", err)
			}
		} else {
			return time.Time{}, 0, fmt.Errorf("load global throttle bucket: %w", err)
		}
	}
	bucketStart = row.BucketStart
	count = row.Count

	if !bucketStart.Equal(current) {
		bucketStart = current
		count = 0
		if err := queries.UpdateGlobalThrottleBucket(ctx, ratelimitsqlc.UpdateGlobalThrottleBucketParams{
			BucketType:  bucketType,
			BucketStart: bucketStart,
			Count:       0,
		}); err != nil {
			return time.Time{}, 0, fmt.Errorf("reset global throttle bucket: %w", err)
		}
	}
	return bucketStart, int(count), nil
}

func (s *PostgresStore) updateBucket(ctx context.Context, queries *ratelimitsqlc.Queries, bucketType string, bucketStart time.Time, count int) error {
	if err := queries.UpdateGlobalThrottleBucket(ctx, ratelimitsqlc.UpdateGlobalThrottleBucketParams{
		BucketType:  bucketType,
		BucketStart: bucketStart,
		Count:       int32(count),
	}); err != nil {
		return fmt.Errorf("update global throttle bucket: %w", err)
	}
	return nil
}
