package client

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

type ClientStoreSuite struct {
	suite.Suite
	store *InMemory
	ctx   context.Context
}

func (s *ClientStoreSuite) SetupTest() {
	s.store = NewInMemory()
	s.ctx = context.Background()
}

func TestClientStoreSuite(t *testing.T) {
	suite.Run(t, new(ClientStoreSuite))
}

func (s *ClientStoreSuite) newClient(tenantID id.TenantID) *models.Client {
	return &models.Client{
		ID:            id.ClientID(uuid.New()),
		TenantID:      tenantID,
		Name:          "Test Client",
		OAuthClientID: uuid.NewString(),
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// TestLookups verifies the store correctly indexes and retrieves clients.
func (s *ClientStoreSuite) TestLookups() {
	s.Run("finds by OAuth client ID after creation", func() {
		client := s.newClient(id.TenantID(uuid.New()))
		client.OAuthClientID = "test-client-id"
		s.Require().NoError(s.store.Create(s.ctx, client))

		found, err := s.store.FindByOAuthClientID(s.ctx, "test-client-id")
		s.Require().NoError(err)
		s.Equal(client.ID, found.ID)
	})

	s.Run("returns ErrNotFound for unknown ID", func() {
		_, err := s.store.FindByID(s.ctx, id.ClientID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})

	s.Run("returns ErrNotFound for unknown OAuth client ID", func() {
		_, err := s.store.FindByOAuthClientID(s.ctx, "nonexistent")
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestTenantIsolation verifies clients are properly scoped to their tenant.
func (s *ClientStoreSuite) TestTenantIsolation() {
	s.Run("scoped lookup rejects wrong tenant", func() {
		tenantA := id.TenantID(uuid.New())
		tenantB := id.TenantID(uuid.New())

		client := s.newClient(tenantA)
		s.Require().NoError(s.store.Create(s.ctx, client))

		// Should find with correct tenant
		found, err := s.store.FindByTenantAndID(s.ctx, tenantA, client.ID)
		s.Require().NoError(err)
		s.Equal(client.ID, found.ID)

		// Should NOT find with wrong tenant
		_, err = s.store.FindByTenantAndID(s.ctx, tenantB, client.ID)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})

	s.Run("count only includes matching tenant", func() {
		tenantA := id.TenantID(uuid.New())
		tenantB := id.TenantID(uuid.New())

		// Create 2 clients for tenant A
		for i := 0; i < 2; i++ {
			s.Require().NoError(s.store.Create(s.ctx, s.newClient(tenantA)))
		}

		// Create 3 clients for tenant B
		for i := 0; i < 3; i++ {
			s.Require().NoError(s.store.Create(s.ctx, s.newClient(tenantB)))
		}

		countA, err := s.store.CountByTenant(s.ctx, tenantA)
		s.Require().NoError(err)
		s.Equal(2, countA)

		countB, err := s.store.CountByTenant(s.ctx, tenantB)
		s.Require().NoError(err)
		s.Equal(3, countB)
	})
}
