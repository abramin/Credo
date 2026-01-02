//go:build integration

package store_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"credo/internal/evidence/registry/models"
	"credo/internal/evidence/registry/store"
	id "credo/pkg/domain"
	"credo/pkg/testutil/containers"
)

type PostgresCacheSuite struct {
	suite.Suite
	postgres *containers.PostgresContainer
	cache    *store.PostgresCache
}

func TestPostgresCacheSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(PostgresCacheSuite))
}

func (s *PostgresCacheSuite) SetupSuite() {
	mgr := containers.GetManager()
	s.postgres = mgr.GetPostgres(s.T())
	s.cache = store.NewPostgresCache(s.postgres.DB, 5*time.Minute, nil)
}

func (s *PostgresCacheSuite) SetupTest() {
	ctx := context.Background()
	err := s.postgres.TruncateTables(ctx, "citizen_cache", "sanctions_cache")
	s.Require().NoError(err)
}

func testNationalID(val string) id.NationalID {
	nid, _ := id.ParseNationalID(val)
	return nid
}

// TestConcurrentCitizenUpsert verifies that concurrent upserts on the same key
// result in last-write-wins semantics without partial updates or corruption.
func (s *PostgresCacheSuite) TestConcurrentCitizenUpsert() {
	ctx := context.Background()
	key := testNationalID("CONCURRENT1")
	const goroutines = 50

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			record := &models.CitizenRecord{
				NationalID:  key.String(),
				FullName:    "User " + string(rune('A'+idx%26)),
				DateOfBirth: "1990-01-01",
				Address:     "Address " + string(rune('A'+idx%26)),
				Valid:       idx%2 == 0,
				Source:      "test",
				CheckedAt:   time.Now(),
			}

			err := s.cache.SaveCitizen(ctx, key, record, false)
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All upserts should succeed (ON CONFLICT DO UPDATE)
	s.Equal(int32(goroutines), successCount.Load(), "all concurrent upserts should succeed")

	// Verify exactly one record exists with consistent data
	found, err := s.cache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(key.String(), found.NationalID)
	// The final state should be one of the written values (last write wins)
	s.NotEmpty(found.FullName)
}

// TestConcurrentSanctionUpsert verifies concurrent upserts for sanctions cache.
func (s *PostgresCacheSuite) TestConcurrentSanctionUpsert() {
	ctx := context.Background()
	key := testNationalID("CONCURRENT2")
	const goroutines = 50

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			record := &models.SanctionsRecord{
				NationalID: key.String(),
				Listed:     idx%2 == 0,
				Source:     "test",
				CheckedAt:  time.Now(),
			}

			err := s.cache.SaveSanction(ctx, key, record)
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(goroutines), successCount.Load(), "all concurrent upserts should succeed")

	found, err := s.cache.FindSanction(ctx, key)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(key.String(), found.NationalID)
}

// TestCacheTTLBoundary verifies that records are correctly expired based on TTL.
func (s *PostgresCacheSuite) TestCacheTTLBoundary() {
	// Create a cache with a very short TTL for testing
	shortTTLCache := store.NewPostgresCache(s.postgres.DB, 1*time.Second, nil)
	ctx := context.Background()
	key := testNationalID("TTLTEST1")

	record := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "TTL Test User",
		DateOfBirth: "1990-01-01",
		Address:     "TTL Test Address",
		Valid:       true,
		Source:      "test",
		CheckedAt:   time.Now(),
	}

	err := shortTTLCache.SaveCitizen(ctx, key, record, false)
	s.Require().NoError(err)

	// Should be found immediately
	found, err := shortTTLCache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)
	s.NotNil(found)

	// Wait for TTL to expire
	time.Sleep(1500 * time.Millisecond)

	// Should not be found after TTL
	_, err = shortTTLCache.FindCitizen(ctx, key, false)
	s.ErrorIs(err, store.ErrNotFound)
}

// TestConcurrentMixedOperations verifies concurrent saves and finds don't interfere.
func (s *PostgresCacheSuite) TestConcurrentMixedOperations() {
	ctx := context.Background()
	const goroutines = 30
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	var saveErrors atomic.Int32
	var findErrors atomic.Int32

	// Launch goroutines that do both saves and finds
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := testNationalID("MIXED" + string(rune('A'+idx%10)))

			for j := 0; j < opsPerGoroutine; j++ {
				record := &models.CitizenRecord{
					NationalID:  key.String(),
					FullName:    "User",
					DateOfBirth: "1990-01-01",
					Address:     "Address",
					Valid:       true,
					Source:      "test",
					CheckedAt:   time.Now(),
				}

				if err := s.cache.SaveCitizen(ctx, key, record, false); err != nil {
					saveErrors.Add(1)
				}

				// Interleave with reads - may or may not find based on timing
				_, err := s.cache.FindCitizen(ctx, key, false)
				if err != nil && err != store.ErrNotFound {
					findErrors.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	// No unexpected errors should occur
	s.Equal(int32(0), saveErrors.Load(), "no save errors expected")
	s.Equal(int32(0), findErrors.Load(), "no unexpected find errors")
}

// TestRegulatedModeIsolation verifies regulated and non-regulated caches are isolated.
func (s *PostgresCacheSuite) TestRegulatedModeIsolation() {
	ctx := context.Background()
	key := testNationalID("REGULATE1")

	nonRegulatedRecord := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "Full Name Visible",
		DateOfBirth: "1990-01-01",
		Address:     "Full Address",
		Valid:       true,
		Source:      "test",
		CheckedAt:   time.Now(),
	}

	regulatedRecord := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "", // PII minimized in regulated mode
		DateOfBirth: "",
		Address:     "",
		Valid:       true,
		Source:      "test",
		CheckedAt:   time.Now(),
	}

	// Save both versions
	err := s.cache.SaveCitizen(ctx, key, nonRegulatedRecord, false)
	s.Require().NoError(err)
	err = s.cache.SaveCitizen(ctx, key, regulatedRecord, true)
	s.Require().NoError(err)

	// Each should be retrieved independently
	foundNonReg, err := s.cache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)
	s.Equal("Full Name Visible", foundNonReg.FullName)

	foundReg, err := s.cache.FindCitizen(ctx, key, true)
	s.Require().NoError(err)
	s.Empty(foundReg.FullName, "regulated mode should have empty PII fields")
}

// TestConcurrentRegulatedUpserts verifies concurrent upserts across regulated modes.
func (s *PostgresCacheSuite) TestConcurrentRegulatedUpserts() {
	ctx := context.Background()
	key := testNationalID("REGCON1")
	const goroutines = 50

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			regulated := idx%2 == 0
			record := &models.CitizenRecord{
				NationalID:  key.String(),
				FullName:    "User",
				DateOfBirth: "1990-01-01",
				Address:     "Address",
				Valid:       true,
				Source:      "test",
				CheckedAt:   time.Now(),
			}

			_ = s.cache.SaveCitizen(ctx, key, record, regulated)
		}(i)
	}

	wg.Wait()

	// Both regulated and non-regulated should exist (separate rows via composite PK)
	_, err := s.cache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)

	_, err = s.cache.FindCitizen(ctx, key, true)
	s.Require().NoError(err)
}
