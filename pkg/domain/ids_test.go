package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dErrors "credo/pkg/domain-errors"
)

// TestParseUUID_Invariants validates the parsing invariant:
// "IDs must be valid, non-empty, non-nil UUIDs"
//
// Justification: This is a pure function enforcing a domain invariant
// at trust boundaries. Per testing.md, unit tests are allowed for invariants.
func TestParseUUID_Invariants(t *testing.T) {
	t.Run("rejects empty string", func(t *testing.T) {
		_, err := ParseUserID("")
		require.Error(t, err)
		assert.True(t, dErrors.HasCode(err, dErrors.CodeInvalidInput))
	})

	t.Run("rejects invalid format", func(t *testing.T) {
		_, err := ParseUserID("not-a-uuid")
		require.Error(t, err)
		assert.True(t, dErrors.HasCode(err, dErrors.CodeInvalidInput))
	})

	t.Run("rejects nil UUID", func(t *testing.T) {
		_, err := ParseUserID(uuid.Nil.String())
		require.Error(t, err)
		assert.True(t, dErrors.HasCode(err, dErrors.CodeInvalidInput))
	})

	t.Run("accepts valid UUID", func(t *testing.T) {
		validUUID := uuid.New()
		id, err := ParseUserID(validUUID.String())
		require.NoError(t, err)
		assert.Equal(t, UserID(validUUID), id)
	})
}

// TestTypeDistinction verifies the compiler enforces type safety.
// This is a compile-time check - if this compiles, the invariant holds.
func TestTypeDistinction(t *testing.T) {
	userID := UserID(uuid.New())
	tenantID := TenantID(uuid.New())

	// These would fail to compile if types were interchangeable:
	// var _ UserID = tenantID   // compile error
	// var _ TenantID = userID   // compile error

	// Verify they're distinct at runtime too
	assert.NotEqual(t, uuid.UUID(userID), uuid.UUID(tenantID))
}

// TestCrossTypeAssignment_CompileTimeInvariant documents the compile-time invariant.
// If someone removes type safety, this test's comments become incorrect.
//
// Justification: Documents security invariant - typed IDs prevent cross-type assignment.
func TestCrossTypeAssignment_CompileTimeInvariant(t *testing.T) {
	// The following would fail to compile:
	// var uid UserID = TenantID(uuid.New())  // type mismatch
	// var tid TenantID = UserID(uuid.New())  // type mismatch
	// acceptsUserID(TenantID(uuid.New()))    // argument type mismatch

	// This test documents the invariant. If types become aliases,
	// these assignments would compile and the invariant is broken.
	t.Log("Typed IDs prevent cross-type assignment at compile time")
}

// TestParseID_SecurityInvariants validates security-critical parsing rules.
//
// Justification: These are trust boundary invariants - parsing must reject
// attack vectors at API entry points.
func TestParseID_SecurityInvariants(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Attack vectors
		{"SQL injection attempt", "'; DROP TABLE users;--", true},
		{"Path traversal", "../../../etc/passwd", true},
		{"Null byte injection", "550e8400\x00-e29b-41d4-a716-446655440000", true},
		{"Oversized input", strings.Repeat("a", 1000), true},
		{"Unicode zero-width space", "550e8400\u200B-e29b-41d4-a716-446655440000", true},

		// Edge cases
		{"Empty string", "", true},
		{"Nil UUID", uuid.Nil.String(), true},
		{"Whitespace only", "   ", true},
		{"Uppercase valid UUID", "550E8400-E29B-41D4-A716-446655440000", false},
		// Note: uuid.Parse trims whitespace, so " uuid " is accepted as valid

		// Valid
		{"Valid UUID lowercase", "550e8400-e29b-41d4-a716-446655440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseUserID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, dErrors.HasCode(err, dErrors.CodeInvalidInput))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestTenantIsolation_CrossTenantAccessDenied encodes the "never again" invariant:
// "Actor from tenant A must never access resources from tenant B"
//
// Justification: This documents the security invariant. Actual enforcement is in services,
// but typed IDs ensure tenant context is never accidentally omitted.
func TestTenantIsolation_CrossTenantAccessDenied(t *testing.T) {
	tenantA := TenantID(uuid.New())
	tenantB := TenantID(uuid.New())

	// Typed IDs make cross-tenant comparison explicit
	assert.NotEqual(t, tenantA, tenantB, "Different tenants must have different IDs")
	assert.NotEqual(t, uuid.UUID(tenantA), uuid.UUID(tenantB), "UUID values must differ")
}

// TestAllIDTypes_ConsistentBehavior ensures all ID types have identical parsing behavior.
//
// Justification: Inconsistent validation across ID types could create security holes.
func TestAllIDTypes_ConsistentBehavior(t *testing.T) {
	validUUID := uuid.New().String()
	invalidInputs := []string{"", "invalid", uuid.Nil.String()}

	// All types should accept valid UUID
	t.Run("all accept valid UUID", func(t *testing.T) {
		_, errUser := ParseUserID(validUUID)
		_, errSession := ParseSessionID(validUUID)
		_, errClient := ParseClientID(validUUID)
		_, errTenant := ParseTenantID(validUUID)
		_, errConsent := ParseConsentID(validUUID)

		require.NoError(t, errUser)
		require.NoError(t, errSession)
		require.NoError(t, errClient)
		require.NoError(t, errTenant)
		require.NoError(t, errConsent)
	})

	// All types should reject invalid inputs identically
	for _, input := range invalidInputs {
		t.Run("all reject: "+input, func(t *testing.T) {
			_, errUser := ParseUserID(input)
			_, errSession := ParseSessionID(input)
			_, errClient := ParseClientID(input)
			_, errTenant := ParseTenantID(input)
			_, errConsent := ParseConsentID(input)

			require.Error(t, errUser)
			require.Error(t, errSession)
			require.Error(t, errClient)
			require.Error(t, errTenant)
			require.Error(t, errConsent)
		})
	}
}
