package domain

import (
	"fmt"
)

// APIVersion represents a valid API version string.
// This is a domain primitive that enforces validity at parse time.
type APIVersion string

// Supported API versions.
const (
	APIVersionV1 APIVersion = "v1"
	// Future versions: APIVersionV2 APIVersion = "v2"
)

// versionOrder defines the ordering of versions for comparison.
// Higher numbers represent newer versions.
var versionOrder = map[APIVersion]int{
	APIVersionV1: 1,
	// APIVersionV2: 2,
}

// ParseAPIVersion validates and returns an APIVersion.
// Returns an error if the version is unknown.
func ParseAPIVersion(s string) (APIVersion, error) {
	v := APIVersion(s)
	if _, ok := versionOrder[v]; !ok {
		return "", fmt.Errorf("unknown API version: %s", s)
	}
	return v, nil
}

// String returns the string representation of the API version.
func (v APIVersion) String() string {
	return string(v)
}

// IsNil returns true if the API version is empty.
func (v APIVersion) IsNil() bool {
	return v == ""
}

// IsAtLeast returns true if this version is >= other.
// Used for forward compatibility checks:
//   - v1 token on v2 route: routeVersion(v2).IsAtLeast(tokenVersion(v1)) = true (OK)
//   - v2 token on v1 route: routeVersion(v1).IsAtLeast(tokenVersion(v2)) = false (REJECTED)
func (v APIVersion) IsAtLeast(other APIVersion) bool {
	thisOrder, thisOK := versionOrder[v]
	otherOrder, otherOK := versionOrder[other]

	// Unknown versions are treated as lower than any known version
	if !thisOK {
		return false
	}
	if !otherOK {
		return true // Any known version is >= unknown
	}

	return thisOrder >= otherOrder
}

// SupportedVersions returns all currently supported API versions.
func SupportedVersions() []APIVersion {
	return []APIVersion{APIVersionV1}
}

// DefaultVersion returns the default API version for new tokens.
func DefaultVersion() APIVersion {
	return APIVersionV1
}
