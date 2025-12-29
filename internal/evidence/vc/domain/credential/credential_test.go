package credential_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"credo/internal/evidence/vc/domain/credential"
	"credo/internal/evidence/vc/domain/shared"
	"credo/internal/evidence/vc/models"
	id "credo/pkg/domain"
)

type CredentialSuite struct {
	suite.Suite
	validID       models.CredentialID
	validType     models.CredentialType
	validSubject  id.UserID
	validIssuer   string
	validIssuedAt shared.IssuedAt
	validClaims   credential.ClaimSet
}

func TestCredentialSuite(t *testing.T) {
	suite.Run(t, new(CredentialSuite))
}

func (s *CredentialSuite) SetupTest() {
	s.validID = models.NewCredentialID()
	s.validType = models.CredentialTypeAgeOver18
	s.validSubject = id.UserID(uuid.New())
	s.validIssuer = "credo"
	s.validIssuedAt = shared.MustIssuedAt(time.Now())
	s.validClaims = credential.NewAgeOver18Claims(true, "national_registry")
}

func (s *CredentialSuite) TestConstructionInvariants() {
	s.Run("rejects empty credential ID", func() {
		_, err := credential.New(
			"",
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "credential_id")
	})

	s.Run("rejects nil subject", func() {
		_, err := credential.New(
			s.validID,
			s.validType,
			id.UserID{}, // nil user ID
			s.validIssuer,
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "subject")
	})

	s.Run("rejects empty issuer", func() {
		_, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			"",
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "issuer")
	})

	s.Run("rejects zero issued_at", func() {
		_, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			shared.IssuedAt{}, // zero value
			s.validClaims,
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "issued_at")
	})

	s.Run("rejects nil claims", func() {
		_, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			nil,
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "claims")
	})

	s.Run("accepts valid inputs", func() {
		cred, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().NoError(err)
		s.NotNil(cred)
		s.Equal(s.validID, cred.ID())
		s.Equal(s.validType, cred.Type())
		s.Equal(s.validSubject, cred.Subject())
		s.Equal(s.validIssuer, cred.Issuer())
		s.False(cred.IsMinimized())
	})
}

func (s *CredentialSuite) TestMinimization() {
	s.Run("returns new credential without mutating original", func() {
		original, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().NoError(err)

		minimized := original.Minimized()

		// Original should be unchanged
		s.False(original.IsMinimized())

		// Minimized should be marked as such
		s.True(minimized.IsMinimized())

		// Both should have same ID and metadata
		s.Equal(original.ID(), minimized.ID())
		s.Equal(original.Type(), minimized.Type())
		s.Equal(original.Subject(), minimized.Subject())
		s.Equal(original.Issuer(), minimized.Issuer())
	})

	s.Run("strips verified_via from AgeOver18 claims", func() {
		claims := credential.NewAgeOver18Claims(true, "national_registry")
		original, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			claims,
		)
		s.Require().NoError(err)

		minimized := original.Minimized()

		// Original claims should have verified_via
		originalMap := original.Claims().ToMap()
		s.Contains(originalMap, "verified_via")

		// Minimized claims should NOT have verified_via
		minimizedMap := minimized.Claims().ToMap()
		s.NotContains(minimizedMap, "verified_via")

		// But should still have is_over_18
		s.Contains(minimizedMap, "is_over_18")
		s.Equal(true, minimizedMap["is_over_18"])
	})
}

func (s *CredentialSuite) TestAccessors() {
	s.Run("all accessors return correct values", func() {
		cred, err := credential.New(
			s.validID,
			s.validType,
			s.validSubject,
			s.validIssuer,
			s.validIssuedAt,
			s.validClaims,
		)
		s.Require().NoError(err)

		s.Equal(s.validID, cred.ID())
		s.Equal(s.validType, cred.Type())
		s.Equal(s.validSubject, cred.Subject())
		s.Equal(s.validIssuer, cred.Issuer())
		s.Equal(s.validIssuedAt, cred.IssuedAt())
		s.NotNil(cred.Claims())
	})
}
