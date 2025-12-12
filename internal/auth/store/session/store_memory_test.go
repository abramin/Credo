package session

import (
	"context"
	"testing"
	"time"

	"credo/internal/auth/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type InMemorySessionStoreSuite struct {
	suite.Suite
	store *InMemorySessionStore
}

func (s *InMemorySessionStoreSuite) SetupTest() {
	s.store = NewInMemorySessionStore()
}

func (s *InMemorySessionStoreSuite) TestSaveAndFind() {
	session := &models.Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		RequestedScope: []string{"openid"},
		Status:         "pending",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	err := s.store.Save(context.Background(), session)
	require.NoError(s.T(), err)

	foundByID, err := s.store.FindByID(context.Background(), session.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), session, foundByID)

	foundByCode, err := s.store.FindByCode(context.Background(), session.Code)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), session, foundByCode)
}

func (s *InMemorySessionStoreSuite) TestFindNotFound() {
	_, err := s.store.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func TestInMemorySessionStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemorySessionStoreSuite))
}
