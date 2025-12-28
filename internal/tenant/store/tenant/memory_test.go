package tenant

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"credo/internal/tenant/models"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"
)

type TenantStoreSuite struct {
	suite.Suite
	store *InMemory
	ctx   context.Context
}

func (s *TenantStoreSuite) SetupTest() {
	s.store = NewInMemory()
	s.ctx = context.Background()
}

func TestTenantStoreSuite(t *testing.T) {
	suite.Run(t, new(TenantStoreSuite))
}

func (s *TenantStoreSuite) newTenant(name string) *models.Tenant {
	return &models.Tenant{
		ID:        id.TenantID(uuid.New()),
		Name:      name,
		Status:    models.TenantStatusActive,
		CreatedAt: time.Now(),
	}
}

// TestCreationAndLookups verifies the store correctly creates and retrieves tenants.
func (s *TenantStoreSuite) TestCreationAndLookups() {
	s.Run("creates and finds tenant by ID", func() {
		tenant := s.newTenant("Test Tenant")
		s.Require().NoError(s.store.CreateIfNameAvailable(s.ctx, tenant))

		found, err := s.store.FindByID(s.ctx, tenant.ID)
		s.Require().NoError(err)
		s.Equal(tenant.Name, found.Name)
	})

	s.Run("returns ErrNotFound for unknown ID", func() {
		_, err := s.store.FindByID(s.ctx, id.TenantID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestNameUniqueness verifies case-insensitive name uniqueness enforcement.
func (s *TenantStoreSuite) TestNameUniqueness() {
	s.Run("rejects duplicate name", func() {
		tenant1 := s.newTenant("Duplicate")
		tenant2 := s.newTenant("Duplicate")
		tenant2.ID = id.TenantID(uuid.New()) // different ID

		s.Require().NoError(s.store.CreateIfNameAvailable(s.ctx, tenant1))

		err := s.store.CreateIfNameAvailable(s.ctx, tenant2)
		s.Require().Error(err)
		s.ErrorIs(err, sentinel.ErrAlreadyUsed)
	})

	s.Run("enforces case-insensitive uniqueness", func() {
		tenant1 := s.newTenant("MyTenant")
		tenant2 := s.newTenant("MYTENANT")
		tenant2.ID = id.TenantID(uuid.New())

		s.Require().NoError(s.store.CreateIfNameAvailable(s.ctx, tenant1))

		err := s.store.CreateIfNameAvailable(s.ctx, tenant2)
		s.Require().Error(err)
		s.ErrorIs(err, sentinel.ErrAlreadyUsed)
	})

	s.Run("finds by name case-insensitively", func() {
		tenant := s.newTenant("CaseSensitive")
		s.Require().NoError(s.store.CreateIfNameAvailable(s.ctx, tenant))

		// Find with different cases
		found, err := s.store.FindByName(s.ctx, "casesensitive")
		s.Require().NoError(err)
		s.Equal(tenant.ID, found.ID)

		found, err = s.store.FindByName(s.ctx, "CASESENSITIVE")
		s.Require().NoError(err)
		s.Equal(tenant.ID, found.ID)
	})
}

// TestUpdates verifies the store correctly persists and validates updates.
func (s *TenantStoreSuite) TestUpdates() {
	s.Run("persists status changes", func() {
		tenant := s.newTenant("Update Test")
		s.Require().NoError(s.store.CreateIfNameAvailable(s.ctx, tenant))

		tenant.Status = models.TenantStatusInactive
		s.Require().NoError(s.store.Update(s.ctx, tenant))

		found, err := s.store.FindByID(s.ctx, tenant.ID)
		s.Require().NoError(err)
		s.Equal(models.TenantStatusInactive, found.Status)
	})

	s.Run("returns ErrNotFound for non-existent tenant", func() {
		tenant := s.newTenant("Nonexistent")

		err := s.store.Update(s.ctx, tenant)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}
