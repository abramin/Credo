// Package shared provides the shared kernel for the Registry bounded context.
//
// The shared kernel contains domain primitives that are used across both the
// Citizen and Sanctions subdomains. These types form the common vocabulary
// for identity evidence within the Registry context.
//
// Domain Purity: This package contains only pure domain types with no I/O,
// no context.Context, and no time.Now() calls. Time is always received as
// a parameter from the application layer.
package shared

import (
	"errors"
	"regexp"
	"time"
)

// NationalID is a validated national identifier used as a lookup key for
// both Citizen and Sanctions registry lookups.
//
// Invariants:
//   - Non-empty
//   - Alphanumeric only (A-Z, 0-9)
//   - Length between 6 and 20 characters
type NationalID struct {
	value string
}

var nationalIDPattern = regexp.MustCompile(`^[A-Z0-9]{6,20}$`)

// ErrInvalidNationalID indicates the national ID failed validation.
var ErrInvalidNationalID = errors.New("invalid national ID: must be 6-20 alphanumeric characters")

// NewNationalID creates a validated NationalID.
// Returns an error if the value doesn't match the required pattern.
func NewNationalID(value string) (NationalID, error) {
	if !nationalIDPattern.MatchString(value) {
		return NationalID{}, ErrInvalidNationalID
	}
	return NationalID{value: value}, nil
}

// MustNationalID creates a NationalID, panicking if invalid.
// Use only in tests or when the value is known to be valid.
func MustNationalID(value string) NationalID {
	id, err := NewNationalID(value)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the national ID value.
func (n NationalID) String() string {
	return n.value
}

// IsZero returns true if this is the zero value (uninitialized).
func (n NationalID) IsZero() bool {
	return n.value == ""
}

// Confidence represents the reliability score of evidence from a provider.
// Range: 0.0 (no confidence) to 1.0 (authoritative source).
//
// Invariants:
//   - Value must be between 0.0 and 1.0 inclusive
type Confidence struct {
	value float64
}

// ErrInvalidConfidence indicates the confidence score is out of range.
var ErrInvalidConfidence = errors.New("invalid confidence: must be between 0.0 and 1.0")

// NewConfidence creates a validated Confidence score.
func NewConfidence(value float64) (Confidence, error) {
	if value < 0.0 || value > 1.0 {
		return Confidence{}, ErrInvalidConfidence
	}
	return Confidence{value: value}, nil
}

// MustConfidence creates a Confidence, panicking if invalid.
func MustConfidence(value float64) Confidence {
	c, err := NewConfidence(value)
	if err != nil {
		panic(err)
	}
	return c
}

// Authoritative returns a Confidence of 1.0 (fully trusted source).
func Authoritative() Confidence {
	return Confidence{value: 1.0}
}

// Value returns the confidence score.
func (c Confidence) Value() float64 {
	return c.value
}

// IsAuthoritative returns true if confidence is 1.0.
func (c Confidence) IsAuthoritative() bool {
	return c.value == 1.0
}

// CheckedAt represents the timestamp when evidence was fetched from a registry.
// This is a value object that encapsulates verification timing.
type CheckedAt struct {
	value time.Time
}

// NewCheckedAt creates a CheckedAt from a time value.
// The time should be provided by the application layer (not called with time.Now() in domain).
func NewCheckedAt(t time.Time) CheckedAt {
	return CheckedAt{value: t}
}

// Time returns the underlying time value.
func (c CheckedAt) Time() time.Time {
	return c.value
}

// IsExpiredAt checks if this check is older than the given TTL relative to 'now'.
// Both 'now' and 'ttl' are provided by the caller to maintain domain purity.
func (c CheckedAt) IsExpiredAt(now time.Time, ttl time.Duration) bool {
	return now.Sub(c.value) > ttl
}

// IsFreshAt checks if this check is still valid given the TTL and current time.
func (c CheckedAt) IsFreshAt(now time.Time, ttl time.Duration) bool {
	return !c.IsExpiredAt(now, ttl)
}

// IsZero returns true if this is the zero value.
func (c CheckedAt) IsZero() bool {
	return c.value.IsZero()
}

// ProviderID identifies the source of evidence.
// This is used to track which registry or provider produced a piece of evidence.
type ProviderID struct {
	value string
}

// NewProviderID creates a ProviderID.
func NewProviderID(value string) ProviderID {
	return ProviderID{value: value}
}

// String returns the provider ID.
func (p ProviderID) String() string {
	return p.value
}

// IsZero returns true if this is the zero value.
func (p ProviderID) IsZero() bool {
	return p.value == ""
}
