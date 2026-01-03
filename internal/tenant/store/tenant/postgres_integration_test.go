//go:build integration

package tenant_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"credo/internal/tenant/models"
	"credo/internal/tenant/store/tenant"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"
	"credo/pkg/testutil/containers"
)

type PostgresStoreSuite struct {
	suite.Suite
	postgres *containers.PostgresContainer
	store    *tenant.PostgresStore
}

func TestPostgresStoreSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(PostgresStoreSuite))
}

func (s *PostgresStoreSuite) SetupSuite() {
	mgr := containers.GetManager()
	s.postgres = mgr.GetPostgres(s.T())
	s.store = tenant.NewPostgres(s.postgres.DB)
}

func (s *PostgresStoreSuite) SetupTest() {
	ctx := context.Background()
	// Truncate in dependency order
	err := s.postgres.TruncateTables(ctx, "users", "clients", "tenants")
	s.Require().NoError(err)
}

func newTestTenant(name string) *models.Tenant {
	now := time.Now()
	return &models.Tenant{
		ID:        id.TenantID(uuid.New()),
		Name:      name,
		Status:    models.TenantStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestConcurrentUniqueNameViolation verifies that concurrent creation attempts
// with the same name result in exactly one success.
func (s *PostgresStoreSuite) TestConcurrentUniqueNameViolation() {
	ctx := context.Background()
	tenantName := "Concurrent Test Tenant " + uuid.NewString()
	const goroutines = 50

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var conflictCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			t := newTestTenant(tenantName)
			err := s.store.CreateIfNameAvailable(ctx, t)
			if err == nil {
				successCount.Add(1)
			} else if errors.Is(err, sentinel.ErrAlreadyUsed) {
				conflictCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Exactly one should succeed
	s.Equal(int32(1), successCount.Load(), "exactly one create should succeed")
	// All others should get conflict error
	s.Equal(int32(goroutines-1), conflictCount.Load(), "all others should get conflict error")

	// Verify only one tenant exists with this name
	found, err := s.store.FindByName(ctx, tenantName)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(tenantName, found.Name)
}

// TestCaseInsensitiveUniqueness verifies that tenant names are unique regardless of case.
func (s *PostgresStoreSuite) TestCaseInsensitiveUniqueness() {
	ctx := context.Background()
	baseName := "CaseTest" + uuid.NewString()

	// Create with mixed case
	t1 := newTestTenant(baseName)
	err := s.store.CreateIfNameAvailable(ctx, t1)
	s.Require().NoError(err)

	// Try to create with different cases - all should fail
	testCases := []string{
		strings.ToUpper(baseName),
		strings.ToLower(baseName),
		strings.Title(strings.ToLower(baseName)),
	}

	for _, name := range testCases {
		t := newTestTenant(name)
		err := s.store.CreateIfNameAvailable(ctx, t)
		s.ErrorIs(err, sentinel.ErrAlreadyUsed, "name %q should conflict with %q", name, baseName)
	}

	// FindByName should work with any case
	for _, name := range testCases {
		found, err := s.store.FindByName(ctx, name)
		s.Require().NoError(err)
		s.Equal(t1.ID, found.ID, "FindByName(%q) should find the same tenant", name)
	}
}

// TestConcurrentDifferentNames verifies concurrent creation of different tenant names.
func (s *PostgresStoreSuite) TestConcurrentDifferentNames() {
	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	var errors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			t := newTestTenant("Tenant " + uuid.NewString())
			if err := s.store.CreateIfNameAvailable(ctx, t); err != nil {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(0), errors.Load(), "no errors expected for unique names")

	// Verify count
	count, err := s.store.Count(ctx)
	s.Require().NoError(err)
	s.Equal(goroutines, count)
}

// TestUpdateAfterConcurrentRead verifies behavior when updates race with reads.
func (s *PostgresStoreSuite) TestUpdateAfterConcurrentRead() {
	ctx := context.Background()

	// Create a tenant
	t := newTestTenant("Update Race Test " + uuid.NewString())
	err := s.store.CreateIfNameAvailable(ctx, t)
	s.Require().NoError(err)

	const goroutines = 50
	var wg sync.WaitGroup
	var readErrors, updateErrors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			if idx%5 == 0 {
				// Update
				updated := &models.Tenant{
					ID:        t.ID,
					Name:      t.Name, // Keep same name to avoid unique constraint issues
					Status:    models.TenantStatusActive,
					UpdatedAt: time.Now(),
				}
				if err := s.store.Update(ctx, updated); err != nil {
					updateErrors.Add(1)
				}
			} else {
				// Read
				if _, err := s.store.FindByID(ctx, t.ID); err != nil {
					readErrors.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(0), readErrors.Load(), "no read errors expected")
	s.Equal(int32(0), updateErrors.Load(), "no update errors expected")
}

// TestConcurrentUpdateSameTenant verifies concurrent updates to the same tenant.
func (s *PostgresStoreSuite) TestConcurrentUpdateSameTenant() {
	ctx := context.Background()

	// Create a tenant
	t := newTestTenant("Concurrent Update Test " + uuid.NewString())
	err := s.store.CreateIfNameAvailable(ctx, t)
	s.Require().NoError(err)

	const goroutines = 50
	var wg sync.WaitGroup
	var errors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			updated := &models.Tenant{
				ID:        t.ID,
				Name:      t.Name,
				Status:    models.TenantStatusActive,
				UpdatedAt: time.Now().Add(time.Duration(idx) * time.Millisecond),
			}
			if err := s.store.Update(ctx, updated); err != nil {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(0), errors.Load(), "all updates should succeed (last write wins)")

	// Verify tenant still exists with valid state
	found, err := s.store.FindByID(ctx, t.ID)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(t.Name, found.Name)
}

// TestNotFoundError verifies proper error handling for non-existent tenants.
func (s *PostgresStoreSuite) TestNotFoundError() {
	ctx := context.Background()

	// FindByID with non-existent ID
	_, err := s.store.FindByID(ctx, id.TenantID(uuid.New()))
	s.ErrorIs(err, sentinel.ErrNotFound)

	// FindByName with non-existent name
	_, err = s.store.FindByName(ctx, "Non Existent Tenant "+uuid.NewString())
	s.ErrorIs(err, sentinel.ErrNotFound)

	// Update non-existent tenant
	t := newTestTenant("Ghost Tenant")
	err = s.store.Update(ctx, t)
	s.ErrorIs(err, sentinel.ErrNotFound)
}

// TestCountUnderConcurrentCreation verifies Count accuracy during concurrent creation.
func (s *PostgresStoreSuite) TestCountUnderConcurrentCreation() {
	ctx := context.Background()
	const goroutines = 30

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := newTestTenant("Count Test " + uuid.NewString())
			_ = s.store.CreateIfNameAvailable(ctx, t)
		}()
	}

	wg.Wait()

	// Final count should equal goroutines
	count, err := s.store.Count(ctx)
	s.Require().NoError(err)
	s.Equal(goroutines, count)
}
