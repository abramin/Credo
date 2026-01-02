//go:build integration

package client_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"credo/internal/tenant/models"
	"credo/internal/tenant/store/client"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"
	"credo/pkg/testutil/containers"
)

type PostgresStoreSuite struct {
	suite.Suite
	postgres *containers.PostgresContainer
	store    *client.PostgresStore
	tenantID id.TenantID
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
	s.store = client.NewPostgres(s.postgres.DB)
}

func (s *PostgresStoreSuite) SetupTest() {
	ctx := context.Background()

	// Truncate in dependency order
	err := s.postgres.TruncateTables(ctx, "users", "clients", "tenants")
	s.Require().NoError(err)

	// Create a tenant for FK constraint
	s.tenantID = id.TenantID(uuid.New())
	_, err = s.postgres.Exec(ctx, `
		INSERT INTO tenants (id, name, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
	`, uuid.UUID(s.tenantID), "Test Tenant "+uuid.NewString())
	s.Require().NoError(err)
}

func (s *PostgresStoreSuite) newTestClient(oauthClientID string) *models.Client {
	now := time.Now()
	return &models.Client{
		ID:            id.ClientID(uuid.New()),
		TenantID:      s.tenantID,
		Name:          "Test Client " + uuid.NewString(),
		OAuthClientID: oauthClientID,
		RedirectURIs:  []string{"https://example.com/callback"},
		AllowedGrants: []models.GrantType{models.GrantTypeAuthorizationCode},
		AllowedScopes: []string{"openid"},
		Status:        models.ClientStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// TestConcurrentOAuthClientIDCollision verifies that concurrent creation with
// the same OAuth client_id results in exactly one success.
func (s *PostgresStoreSuite) TestConcurrentOAuthClientIDCollision() {
	ctx := context.Background()
	oauthClientID := "client-" + uuid.NewString()
	const goroutines = 50

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var conflictCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			c := s.newTestClient(oauthClientID)
			err := s.store.Create(ctx, c)
			if err == nil {
				successCount.Add(1)
			} else if errors.Is(err, sentinel.ErrAlreadyUsed) {
				conflictCount.Add(1)
			}
		}()
	}

	wg.Wait()

	s.Equal(int32(1), successCount.Load(), "exactly one create should succeed")
	s.Equal(int32(goroutines-1), conflictCount.Load(), "all others should get conflict error")

	// Verify only one client with this OAuth client_id
	found, err := s.store.FindByOAuthClientID(ctx, oauthClientID)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(oauthClientID, found.OAuthClientID)
}

// TestTenantIsolation verifies that FindByTenantAndID respects tenant boundaries.
func (s *PostgresStoreSuite) TestTenantIsolation() {
	ctx := context.Background()

	// Create another tenant
	otherTenantID := id.TenantID(uuid.New())
	_, err := s.postgres.Exec(ctx, `
		INSERT INTO tenants (id, name, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
	`, uuid.UUID(otherTenantID), "Other Tenant "+uuid.NewString())
	s.Require().NoError(err)

	// Create a client under the first tenant
	c := s.newTestClient("isolated-client-" + uuid.NewString())
	err = s.store.Create(ctx, c)
	s.Require().NoError(err)

	// Should find by correct tenant
	found, err := s.store.FindByTenantAndID(ctx, s.tenantID, c.ID)
	s.Require().NoError(err)
	s.Equal(c.ID, found.ID)

	// Should NOT find by other tenant
	_, err = s.store.FindByTenantAndID(ctx, otherTenantID, c.ID)
	s.ErrorIs(err, sentinel.ErrNotFound)

	// FindByID (without tenant filter) should still work
	found, err = s.store.FindByID(ctx, c.ID)
	s.Require().NoError(err)
	s.Equal(c.ID, found.ID)
}

// TestConcurrentDifferentClients verifies concurrent creation of different clients.
func (s *PostgresStoreSuite) TestConcurrentDifferentClients() {
	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	var errors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			c := s.newTestClient("client-" + uuid.NewString())
			if err := s.store.Create(ctx, c); err != nil {
				errors.Add(1)
			}
		}()
	}

	wg.Wait()

	s.Equal(int32(0), errors.Load(), "no errors expected for unique client IDs")

	// Verify count
	count, err := s.store.CountByTenant(ctx, s.tenantID)
	s.Require().NoError(err)
	s.Equal(goroutines, count)
}

// TestConcurrentUpdateSameClient verifies concurrent updates to the same client.
func (s *PostgresStoreSuite) TestConcurrentUpdateSameClient() {
	ctx := context.Background()

	// Create a client
	c := s.newTestClient("update-test-" + uuid.NewString())
	err := s.store.Create(ctx, c)
	s.Require().NoError(err)

	const goroutines = 50
	var wg sync.WaitGroup
	var errors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			updated := &models.Client{
				ID:            c.ID,
				TenantID:      c.TenantID,
				Name:          "Updated " + uuid.NewString(),
				OAuthClientID: c.OAuthClientID,
				RedirectURIs:  c.RedirectURIs,
				AllowedGrants: c.AllowedGrants,
				AllowedScopes: c.AllowedScopes,
				Status:        models.ClientStatusActive,
				UpdatedAt:     time.Now().Add(time.Duration(idx) * time.Millisecond),
			}
			if err := s.store.Update(ctx, updated); err != nil {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(0), errors.Load(), "all updates should succeed (last write wins)")

	// Verify client still exists with valid state
	found, err := s.store.FindByID(ctx, c.ID)
	s.Require().NoError(err)
	s.NotNil(found)
	s.Equal(c.OAuthClientID, found.OAuthClientID)
}

// TestConcurrentCreateAndFind verifies concurrent create and find operations.
func (s *PostgresStoreSuite) TestConcurrentCreateAndFind() {
	ctx := context.Background()

	// Pre-create some clients
	preCreated := make([]*models.Client, 10)
	for i := 0; i < 10; i++ {
		preCreated[i] = s.newTestClient("precreated-" + uuid.NewString())
		err := s.store.Create(ctx, preCreated[i])
		s.Require().NoError(err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	var createErrors, findErrors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			if idx%2 == 0 {
				// Create new
				c := s.newTestClient("concurrent-" + uuid.NewString())
				if err := s.store.Create(ctx, c); err != nil {
					createErrors.Add(1)
				}
			} else {
				// Find existing
				existing := preCreated[idx%10]
				_, err := s.store.FindByOAuthClientID(ctx, existing.OAuthClientID)
				if err != nil {
					findErrors.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	s.Equal(int32(0), createErrors.Load(), "no create errors expected")
	s.Equal(int32(0), findErrors.Load(), "no find errors expected")
}

// TestCountByTenantAccuracy verifies count accuracy under concurrent operations.
func (s *PostgresStoreSuite) TestCountByTenantAccuracy() {
	ctx := context.Background()
	const goroutines = 30

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := s.newTestClient("count-test-" + uuid.NewString())
			_ = s.store.Create(ctx, c)
		}()
	}

	wg.Wait()

	count, err := s.store.CountByTenant(ctx, s.tenantID)
	s.Require().NoError(err)
	s.Equal(goroutines, count)
}

// TestNotFoundError verifies proper error handling for non-existent clients.
func (s *PostgresStoreSuite) TestNotFoundError() {
	ctx := context.Background()

	// FindByID with non-existent ID
	_, err := s.store.FindByID(ctx, id.ClientID(uuid.New()))
	s.ErrorIs(err, sentinel.ErrNotFound)

	// FindByOAuthClientID with non-existent ID
	_, err = s.store.FindByOAuthClientID(ctx, "non-existent-"+uuid.NewString())
	s.ErrorIs(err, sentinel.ErrNotFound)

	// FindByTenantAndID with non-existent client
	_, err = s.store.FindByTenantAndID(ctx, s.tenantID, id.ClientID(uuid.New()))
	s.ErrorIs(err, sentinel.ErrNotFound)

	// Update non-existent client
	c := s.newTestClient("ghost-" + uuid.NewString())
	err = s.store.Update(ctx, c)
	s.ErrorIs(err, sentinel.ErrNotFound)
}
