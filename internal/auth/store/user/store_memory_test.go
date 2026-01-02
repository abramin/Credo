package user

import (
	"context"
	"testing"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// AGENTS.MD JUSTIFICATION: User store invariants (lookup, delete, ErrNotFound)
// are validated here to protect service behavior outside feature coverage.
type InMemoryUserStoreSuite struct {
	suite.Suite
	store *InMemoryUserStore
}

func (s *InMemoryUserStoreSuite) SetupTest() {
	s.store = New()
}

func TestInMemoryUserStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryUserStoreSuite))
}

// TestLookupBehavior tests user retrieval by ID and email.
func (s *InMemoryUserStoreSuite) TestLookupBehavior() {
	s.Run("returns user by ID when exists", func() {
		store := New()
		user := &models.User{
			ID:        id.UserID(uuid.New()),
			Email:     "jane.doe@example.com",
			FirstName: "Jane",
			LastName:  "Doe",
			Verified:  false,
		}
		s.Require().NoError(store.Save(context.Background(), user))

		found, err := store.FindByID(context.Background(), user.ID)
		s.Require().NoError(err)
		s.Equal(user, found)
	})

	s.Run("returns user by email when exists", func() {
		store := New()
		user := &models.User{
			ID:        id.UserID(uuid.New()),
			Email:     "email.lookup@example.com",
			FirstName: "Email",
			LastName:  "Lookup",
		}
		s.Require().NoError(store.Save(context.Background(), user))

		found, err := store.FindByEmail(context.Background(), user.Email)
		s.Require().NoError(err)
		s.Equal(user, found)
	})

	s.Run("returns ErrNotFound when user ID does not exist", func() {
		_, err := s.store.FindByID(context.Background(), id.UserID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})

	s.Run("returns ErrNotFound when email does not exist", func() {
		_, err := s.store.FindByEmail(context.Background(), "missing@example.com")
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestGDPRDeletion tests user deletion for GDPR compliance.
func (s *InMemoryUserStoreSuite) TestGDPRDeletion() {
	s.Run("deletes user and makes them unfindable", func() {
		store := New()
		user := &models.User{
			ID:        id.UserID(uuid.New()),
			Email:     "delete.me@example.com",
			FirstName: "Delete",
			LastName:  "Me",
		}
		s.Require().NoError(store.Save(context.Background(), user))

		s.Require().NoError(store.Delete(context.Background(), user.ID))

		_, err := store.FindByID(context.Background(), user.ID)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})

	s.Run("returns ErrNotFound when deleting non-existent user", func() {
		err := s.store.Delete(context.Background(), id.UserID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}
