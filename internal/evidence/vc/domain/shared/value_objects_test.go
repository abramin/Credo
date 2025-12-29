package shared_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"credo/internal/evidence/vc/domain/shared"
)

type ValueObjectsSuite struct {
	suite.Suite
}

func TestValueObjectsSuite(t *testing.T) {
	suite.Run(t, new(ValueObjectsSuite))
}

func (s *ValueObjectsSuite) TestIssuedAtConstruction() {
	s.Run("rejects zero time", func() {
		_, err := shared.NewIssuedAt(time.Time{})
		s.Require().Error(err)
		s.ErrorIs(err, shared.ErrInvalidIssuedAt)
	})

	s.Run("accepts valid time", func() {
		now := time.Now()
		issuedAt, err := shared.NewIssuedAt(now)
		s.Require().NoError(err)
		s.Equal(now, issuedAt.Time())
	})

	s.Run("accepts past time", func() {
		past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		issuedAt, err := shared.NewIssuedAt(past)
		s.Require().NoError(err)
		s.Equal(past, issuedAt.Time())
	})

	s.Run("accepts future time", func() {
		future := time.Now().Add(24 * time.Hour)
		issuedAt, err := shared.NewIssuedAt(future)
		s.Require().NoError(err)
		s.Equal(future, issuedAt.Time())
	})
}

func (s *ValueObjectsSuite) TestIssuedAtMust() {
	s.Run("panics on zero time", func() {
		s.Panics(func() {
			shared.MustIssuedAt(time.Time{})
		})
	})

	s.Run("returns value on valid time", func() {
		now := time.Now()
		s.NotPanics(func() {
			issuedAt := shared.MustIssuedAt(now)
			s.Equal(now, issuedAt.Time())
		})
	})
}

func (s *ValueObjectsSuite) TestIssuedAtIsZero() {
	s.Run("zero value IssuedAt is zero", func() {
		var issuedAt shared.IssuedAt
		s.True(issuedAt.IsZero())
	})

	s.Run("valid IssuedAt is not zero", func() {
		issuedAt := shared.MustIssuedAt(time.Now())
		s.False(issuedAt.IsZero())
	})
}

func (s *ValueObjectsSuite) TestExpiresAtConstruction() {
	s.Run("rejects zero time", func() {
		_, err := shared.NewExpiresAt(time.Time{})
		s.Require().Error(err)
		s.ErrorIs(err, shared.ErrInvalidExpiresAt)
	})

	s.Run("accepts valid time", func() {
		future := time.Now().Add(24 * time.Hour)
		expiresAt, err := shared.NewExpiresAt(future)
		s.Require().NoError(err)
		s.Equal(future, expiresAt.Time())
	})
}

func (s *ValueObjectsSuite) TestExpiresAtAfterConstruction() {
	s.Run("rejects zero time", func() {
		issuedAt := shared.MustIssuedAt(time.Now())
		_, err := shared.NewExpiresAtAfter(time.Time{}, issuedAt)
		s.Require().Error(err)
		s.ErrorIs(err, shared.ErrInvalidExpiresAt)
	})

	s.Run("rejects expiration before issuance", func() {
		now := time.Now()
		issuedAt := shared.MustIssuedAt(now)
		beforeIssuance := now.Add(-1 * time.Hour)

		_, err := shared.NewExpiresAtAfter(beforeIssuance, issuedAt)
		s.Require().Error(err)
		s.ErrorIs(err, shared.ErrExpiresBeforeIssued)
	})

	s.Run("rejects expiration equal to issuance", func() {
		now := time.Now()
		issuedAt := shared.MustIssuedAt(now)

		_, err := shared.NewExpiresAtAfter(now, issuedAt)
		s.Require().Error(err)
		s.ErrorIs(err, shared.ErrExpiresBeforeIssued)
	})

	s.Run("accepts expiration after issuance", func() {
		now := time.Now()
		issuedAt := shared.MustIssuedAt(now)
		afterIssuance := now.Add(24 * time.Hour)

		expiresAt, err := shared.NewExpiresAtAfter(afterIssuance, issuedAt)
		s.Require().NoError(err)
		s.Equal(afterIssuance, expiresAt.Time())
	})
}

func (s *ValueObjectsSuite) TestNoExpiration() {
	s.Run("returns zero ExpiresAt for permanent credentials", func() {
		exp := shared.NoExpiration()
		s.True(exp.IsZero())
	})
}

func (s *ValueObjectsSuite) TestExpiresAtIsZero() {
	s.Run("zero value ExpiresAt is zero", func() {
		var expiresAt shared.ExpiresAt
		s.True(expiresAt.IsZero())
	})

	s.Run("NoExpiration is zero", func() {
		expiresAt := shared.NoExpiration()
		s.True(expiresAt.IsZero())
	})

	s.Run("valid ExpiresAt is not zero", func() {
		expiresAt, _ := shared.NewExpiresAt(time.Now().Add(time.Hour))
		s.False(expiresAt.IsZero())
	})
}

func (s *ValueObjectsSuite) TestExpiresAtIsExpiredAt() {
	s.Run("permanent credential never expires", func() {
		exp := shared.NoExpiration()
		now := time.Now()
		farFuture := now.Add(100 * 365 * 24 * time.Hour)

		s.False(exp.IsExpiredAt(now))
		s.False(exp.IsExpiredAt(farFuture))
	})

	s.Run("not expired before expiration time", func() {
		expirationTime := time.Now().Add(24 * time.Hour)
		exp, _ := shared.NewExpiresAt(expirationTime)
		now := time.Now()

		s.False(exp.IsExpiredAt(now))
	})

	s.Run("expired after expiration time", func() {
		expirationTime := time.Now().Add(-1 * time.Hour) // already passed
		exp, _ := shared.NewExpiresAt(expirationTime)
		now := time.Now()

		s.True(exp.IsExpiredAt(now))
	})

	s.Run("not expired at exact expiration time", func() {
		expirationTime := time.Now().Add(time.Hour)
		exp, _ := shared.NewExpiresAt(expirationTime)

		// At the exact expiration time, it's not yet expired (uses After, not Before)
		s.False(exp.IsExpiredAt(expirationTime))
	})

	s.Run("expired one nanosecond after expiration time", func() {
		expirationTime := time.Now().Add(time.Hour)
		exp, _ := shared.NewExpiresAt(expirationTime)

		afterExpiration := expirationTime.Add(time.Nanosecond)
		s.True(exp.IsExpiredAt(afterExpiration))
	})
}
