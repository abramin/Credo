package refreshtoken

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

type InMemoryRefreshTokenStoreSuite struct {
	suite.Suite
	store *InMemoryRefreshTokenStore
}

func (s *InMemoryRefreshTokenStoreSuite) SetupTest() {
	s.store = NewInMemoryRefreshTokenStore()
}

func (s *InMemoryRefreshTokenStoreSuite) TestCreateAndFind() {
	sessionID := uuid.New()
	record := &models.RefreshTokenRecord{
		ID:        uuid.New(),
		Token:     "ref_123",
		SessionID: sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := s.store.Create(context.Background(), record)
	require.NoError(s.T(), err)

	foundByID, err := s.store.FindBySessionID(context.Background(), sessionID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), record, foundByID)
}

func (s *InMemoryRefreshTokenStoreSuite) TestFindNotFound() {
	_, err := s.store.FindBySessionID(context.Background(), uuid.New())
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func (s *InMemoryRefreshTokenStoreSuite) TestDeleteSessionsByUser() {
	sessionID := uuid.New()
	otherSessionID := uuid.New()
	matching := &models.RefreshTokenRecord{ID: uuid.New(), Token: "ref_match", SessionID: sessionID}
	other := &models.RefreshTokenRecord{ID: uuid.New(), Token: "ref_other", SessionID: otherSessionID}

	require.NoError(s.T(), s.store.Create(context.Background(), matching))
	require.NoError(s.T(), s.store.Create(context.Background(), other))

	err := s.store.DeleteBySessionID(context.Background(), sessionID)
	require.NoError(s.T(), err)

	_, err = s.store.FindBySessionID(context.Background(), matching.SessionID)
	assert.ErrorIs(s.T(), err, ErrNotFound)

	fetchedOther, err := s.store.FindBySessionID(context.Background(), other.SessionID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), other, fetchedOther)

	err = s.store.DeleteBySessionID(context.Background(), sessionID)
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

func TestInMemoryRefreshTokenStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryRefreshTokenStoreSuite))
}
