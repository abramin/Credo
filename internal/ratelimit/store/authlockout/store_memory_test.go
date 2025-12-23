package authlockout

import (
	"context"
	"testing"
	"time"

	"credo/internal/ratelimit/models"
	"credo/pkg/platform/middleware/requesttime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InMemoryAuthLockoutStoreSuite struct {
	suite.Suite
	store *InMemoryAuthLockoutStore
}

func TestInMemoryAuthLockoutStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryAuthLockoutStoreSuite))
}

func (s *InMemoryAuthLockoutStoreSuite) SetupTest() {
	s.store = New()
}

func (s *InMemoryAuthLockoutStoreSuite) TestGet() {
	ctx := context.Background()

	s.Run("missing identifier returns nil without error", func() {
		record, err := s.store.Get(ctx, "unknown-id")
		s.NoError(err)
		s.Nil(record)
	})

	s.Run("existing record is returned without mutation", func() {
		// Setup: create a record via RecordFailure
		fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		ctx := requesttime.WithTime(context.Background(), fixedTime)
		identifier := "test-user"

		_, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)

		// Get should return the record without mutating it
		record, err := s.store.Get(ctx, identifier)
		s.NoError(err)
		s.NotNil(record)
		s.Equal(identifier, record.Identifier)
		s.Equal(1, record.FailureCount)
		s.Equal(fixedTime, record.LastFailureAt)
	})
}

func (s *InMemoryAuthLockoutStoreSuite) TestRecordFailure() {
	s.Run("first failure creates record with counters initialized to 1", func() {
		fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		ctx := requesttime.WithTime(context.Background(), fixedTime)
		identifier := "new-user"

		record, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)
		s.NotNil(record)

		s.Equal(identifier, record.Identifier)
		s.Equal(1, record.FailureCount)
		s.Equal(1, record.DailyFailures)
		s.Equal(fixedTime, record.LastFailureAt)
		s.Nil(record.LockedUntil)
		s.False(record.RequiresCaptcha)
	})

	s.Run("subsequent failures increment counters", func() {
		firstTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		secondTime := time.Date(2024, 6, 15, 12, 1, 0, 0, time.UTC)
		identifier := "repeat-offender"

		// First failure
		ctx1 := requesttime.WithTime(context.Background(), firstTime)
		record1, err := s.store.RecordFailure(ctx1, identifier)
		s.NoError(err)
		s.Equal(1, record1.FailureCount)
		s.Equal(firstTime, record1.LastFailureAt)

		// Second failure - different time
		ctx2 := requesttime.WithTime(context.Background(), secondTime)
		record2, err := s.store.RecordFailure(ctx2, identifier)
		s.NoError(err)
		s.Equal(2, record2.FailureCount)
		s.Equal(2, record2.DailyFailures)
		s.Equal(secondTime, record2.LastFailureAt)
	})
}

func (s *InMemoryAuthLockoutStoreSuite) TestClear() {
	ctx := context.Background()

	s.Run("clearing existing record removes it", func() {
		identifier := "to-be-cleared"

		// Create record
		_, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)

		// Verify it exists
		record, err := s.store.Get(ctx, identifier)
		s.NoError(err)
		s.NotNil(record)

		// Clear it
		err = s.store.Clear(ctx, identifier)
		s.NoError(err)

		// Verify it's gone
		record, err = s.store.Get(ctx, identifier)
		s.NoError(err)
		s.Nil(record)
	})

	s.Run("clearing missing record is no-op", func() {
		err := s.store.Clear(ctx, "never-existed")
		s.NoError(err)
	})
}

func (s *InMemoryAuthLockoutStoreSuite) TestIsLocked() {
	ctx := context.Background()

	s.Run("unlocked when no record exists", func() {
		locked, lockedUntil, err := s.store.IsLocked(ctx, "unknown")
		s.NoError(err)
		assert.False(s.T(), locked)
		assert.Nil(s.T(), lockedUntil)
	})

	s.Run("locked when LockedUntil is in the future", func() {
		identifier := "locked-user"
		futureTime := time.Now().Add(10 * time.Minute)

		// Create and update record with lock
		_, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)

		record, _ := s.store.Get(ctx, identifier)
		record.LockedUntil = &futureTime
		err = s.store.Update(ctx, record)
		s.NoError(err)

		// Check lock status
		locked, lockedUntil, err := s.store.IsLocked(ctx, identifier)
		s.NoError(err)
		s.True(locked)
		s.Equal(futureTime, *lockedUntil)
	})

	s.Run("unlocked when LockedUntil is in the past", func() {
		identifier := "expired-lock-user"
		pastTime := time.Now().Add(-10 * time.Minute)

		// Create and update record with expired lock
		_, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)

		record, _ := s.store.Get(ctx, identifier)
		record.LockedUntil = &pastTime
		err = s.store.Update(ctx, record)
		s.NoError(err)

		// Check lock status - should be unlocked since lock expired
		locked, _, err := s.store.IsLocked(ctx, identifier)
		s.NoError(err)
		assert.False(s.T(), locked)
	})
}

func (s *InMemoryAuthLockoutStoreSuite) TestUpdate() {
	ctx := context.Background()

	s.Run("update modifies existing record", func() {
		identifier := "updatable-user"

		// Create record
		_, err := s.store.RecordFailure(ctx, identifier)
		s.NoError(err)

		// Update with new values
		updatedRecord := &models.AuthLockout{
			Identifier:      identifier,
			FailureCount:    5,
			DailyFailures:   10,
			RequiresCaptcha: true,
		}
		err = s.store.Update(ctx, updatedRecord)
		s.NoError(err)

		// Verify update took effect
		record, err := s.store.Get(ctx, identifier)
		s.NoError(err)
		s.Equal(5, record.FailureCount)
		s.Equal(10, record.DailyFailures)
		s.True(record.RequiresCaptcha)
	})
}
