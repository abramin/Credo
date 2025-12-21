//go:build go1.18

package domain

import (
	"testing"
	"unicode/utf8"
)

// FuzzParseUserID tests that parsing never panics on arbitrary input
// and always returns either a valid ID or an error.
//
// Justification: Trust boundary functions must handle arbitrary input safely.
// Fuzz tests verify no panics and consistent invariants.
func FuzzParseUserID(f *testing.F) {
	// Seed corpus with interesting inputs
	f.Add("")
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Add("not-a-uuid")
	f.Add("'; DROP TABLE users;--")
	f.Add(string([]byte{0x00, 0x01, 0x02}))
	f.Add("550e8400-e29b-41d4-a716-446655440000\x00suffix")

	f.Fuzz(func(t *testing.T, input string) {
		id, err := ParseUserID(input)

		// Invariant 1: No panics (implicit - test would fail)

		// Invariant 2: Either valid ID or error, never both
		if err == nil {
			// Valid ID must round-trip (including nil UUIDs)
			roundTrip, err2 := ParseUserID(id.String())
			if err2 != nil {
				t.Errorf("Valid ID failed round-trip: %v", err2)
			}
			if roundTrip != id {
				t.Error("Round-trip changed ID value")
			}
		}

		// Invariant 3: Non-UTF8 input must be rejected
		if !utf8.ValidString(input) && err == nil {
			t.Error("Non-UTF8 input was accepted")
		}
	})
}

// FuzzParseAllIDs ensures all ID types have consistent behavior.
//
// Justification: Inconsistent validation across ID types could create security holes.
func FuzzParseAllIDs(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("")
	f.Add("invalid")

	f.Fuzz(func(t *testing.T, input string) {
		// All parse functions should behave consistently
		_, errUser := ParseUserID(input)
		_, errSession := ParseSessionID(input)
		_, errClient := ParseClientID(input)
		_, errTenant := ParseTenantID(input)
		_, errConsent := ParseConsentID(input)

		// If one accepts, all should accept (same underlying validation)
		if errUser == nil {
			if errSession != nil || errClient != nil || errTenant != nil || errConsent != nil {
				t.Error("Inconsistent parsing across ID types")
			}
		}

		// If one rejects, all should reject
		if errUser != nil {
			if errSession == nil || errClient == nil || errTenant == nil || errConsent == nil {
				t.Error("Inconsistent rejection across ID types")
			}
		}
	})
}
