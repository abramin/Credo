package authlockout

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type InMemoryAuthLockoutStoreSuite struct {
	suite.Suite
	store *InMemoryAuthLockoutStore
	ctx   context.Context
}

func TestInMemoryAuthLockoutStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryAuthLockoutStoreSuite))
}

func (s *InMemoryAuthLockoutStoreSuite) SetupTest() {
	s.store = New()
	s.ctx = context.Background()
}

func (s *InMemoryAuthLockoutStoreSuite) TestGet() {
	s.T().Skip("TODO: add contract-focused tests for initial missing records and existing lockout state")
}

func (s *InMemoryAuthLockoutStoreSuite) TestRecordFailure() {
	s.T().Skip("TODO: add tests for first failure, increment behavior, and timestamp updates")
}

func (s *InMemoryAuthLockoutStoreSuite) TestClear() {
	s.T().Skip("TODO: add tests for clearing existing and non-existent lockouts")
}

func (s *InMemoryAuthLockoutStoreSuite) TestIsLocked() {
	s.T().Skip("TODO: add tests for locked, unlocked, and expired lockout states")
}
