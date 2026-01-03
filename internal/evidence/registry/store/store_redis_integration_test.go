//go:build integration

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"credo/internal/evidence/registry/models"
	"credo/internal/evidence/registry/store"
	"credo/pkg/testutil/containers"
)

type RedisCacheSuite struct {
	suite.Suite
	redis *containers.RedisContainer
	cache *store.RedisCache
}

func TestRedisCacheSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(RedisCacheSuite))
}

func (s *RedisCacheSuite) SetupSuite() {
	mgr := containers.GetManager()
	s.redis = mgr.GetRedis(s.T())
	s.cache = store.NewRedisCache(s.redis.Client, 5*time.Minute, nil)
}

func (s *RedisCacheSuite) SetupTest() {
	ctx := context.Background()
	err := s.redis.FlushAll(ctx)
	s.Require().NoError(err)
}

func (s *RedisCacheSuite) TestCitizenRoundTrip() {
	ctx := context.Background()
	key := testNationalID("REDISC1")
	now := time.Now()

	record := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "Redis Citizen",
		DateOfBirth: "1990-01-01",
		Address:     "123 Test St",
		Valid:       true,
		CheckedAt:   now,
	}

	err := s.cache.SaveCitizen(ctx, key, record, false)
	s.Require().NoError(err)

	found, err := s.cache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)
	s.Equal(record.NationalID, found.NationalID)
	s.Equal(record.FullName, found.FullName)
	s.Equal(record.DateOfBirth, found.DateOfBirth)
	s.Equal(record.Address, found.Address)
	s.Equal(record.Valid, found.Valid)
}

func (s *RedisCacheSuite) TestSanctionsRoundTrip() {
	ctx := context.Background()
	key := testNationalID("REDISS1")
	now := time.Now()

	record := &models.SanctionsRecord{
		NationalID: key.String(),
		Listed:     true,
		Source:     "test-source",
		CheckedAt:  now,
	}

	err := s.cache.SaveSanction(ctx, key, record)
	s.Require().NoError(err)

	found, err := s.cache.FindSanction(ctx, key)
	s.Require().NoError(err)
	s.Equal(record.NationalID, found.NationalID)
	s.Equal(record.Listed, found.Listed)
	s.Equal(record.Source, found.Source)
}

func (s *RedisCacheSuite) TestCitizenRegulatedIsolation() {
	ctx := context.Background()
	key := testNationalID("REDISR1")
	now := time.Now()

	nonRegulated := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "Full Name",
		DateOfBirth: "1990-01-01",
		Address:     "Full Address",
		Valid:       true,
		CheckedAt:   now,
	}
	regulated := &models.CitizenRecord{
		NationalID:  key.String(),
		FullName:    "",
		DateOfBirth: "",
		Address:     "",
		Valid:       true,
		CheckedAt:   now,
	}

	err := s.cache.SaveCitizen(ctx, key, nonRegulated, false)
	s.Require().NoError(err)
	err = s.cache.SaveCitizen(ctx, key, regulated, true)
	s.Require().NoError(err)

	foundNonReg, err := s.cache.FindCitizen(ctx, key, false)
	s.Require().NoError(err)
	s.Equal("Full Name", foundNonReg.FullName)

	foundReg, err := s.cache.FindCitizen(ctx, key, true)
	s.Require().NoError(err)
	s.Empty(foundReg.FullName)
}

func (s *RedisCacheSuite) TestCitizenMissReturnsErrNotFound() {
	ctx := context.Background()
	key := testNationalID("MISSED1")
	_, err := s.cache.FindCitizen(ctx, key, false)
	s.ErrorIs(err, store.ErrNotFound)
}

func (s *RedisCacheSuite) TestTTLEviction() {
	ctx := context.Background()
	key := testNationalID("TTLRED1")
	shortTTLCache := store.NewRedisCache(s.redis.Client, 50*time.Millisecond, nil)

	record := &models.CitizenRecord{
		NationalID: key.String(),
		Valid:      true,
		CheckedAt:  time.Now(),
	}

	err := shortTTLCache.SaveCitizen(ctx, key, record, false)
	s.Require().NoError(err)

	time.Sleep(90 * time.Millisecond)

	_, err = shortTTLCache.FindCitizen(ctx, key, false)
	s.ErrorIs(err, store.ErrNotFound)
}
