package device

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// DeviceServiceSuite tests the device binding and user-agent parsing functionality.
// AGENTS.MD JUSTIFICATION: Fingerprint stability and user-agent parsing are internal
// invariants not exposed via E2E tests; deterministic hashing is a pure function contract.
type DeviceServiceSuite struct {
	suite.Suite
	svc *Service
}

func (s *DeviceServiceSuite) SetupTest() {
	s.svc = NewService(true)
}

func TestDeviceServiceSuite(t *testing.T) {
	suite.Run(t, new(DeviceServiceSuite))
}

// TestUserAgentParsing tests the user-agent string parsing for device display names.
func (s *DeviceServiceSuite) TestUserAgentParsing() {
	s.Run("empty user agent returns unknown device", func() {
		result := ParseUserAgent("")
		s.Equal("Unknown Device", result)
	})

	s.Run("chrome on desktop includes browser and OS", func() {
		userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		result := ParseUserAgent(userAgent)
		s.Contains(result, "Chrome")
		s.Contains(result, "on")
		s.NotContains(result, "  ")
	})

	s.Run("safari on iphone includes platform", func() {
		userAgent := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
		result := ParseUserAgent(userAgent)
		s.Contains(result, "on")
		s.Contains(result, "iPhone")
	})

	s.Run("firefox on linux includes browser and OS", func() {
		userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0"
		result := ParseUserAgent(userAgent)
		s.Contains(result, "Firefox")
		s.Contains(result, "on")
	})

	s.Run("unknown user agent returns formatted string", func() {
		result := ParseUserAgent("Unknown/1.0")
		s.Contains(result, "on")
		s.NotEmpty(result)
	})

	s.Run("result has no leading or trailing whitespace", func() {
		userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
		result := ParseUserAgent(userAgent)
		s.Equal(result, strings.TrimSpace(result))
	})
}

// TestFingerprintStability tests that fingerprints are deterministic and stable
// across minor version changes but sensitive to major changes.
func (s *DeviceServiceSuite) TestFingerprintStability() {
	s.Run("disabled service returns empty fingerprint", func() {
		disabled := NewService(false)
		fp := disabled.ComputeFingerprint("Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0")
		s.Empty(fp)
	})

	s.Run("same user agent yields deterministic fingerprint", func() {
		ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

		fp1 := s.svc.ComputeFingerprint(ua)
		fp2 := s.svc.ComputeFingerprint(ua)

		s.Equal(fp1, fp2)
		s.Len(fp1, 64) // SHA-256 hex
	})

	s.Run("minor version changes do not affect fingerprint", func() {
		ua1 := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.109 Safari/537.36"
		ua2 := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.224 Safari/537.36"

		fp1 := s.svc.ComputeFingerprint(ua1)
		fp2 := s.svc.ComputeFingerprint(ua2)

		s.Equal(fp1, fp2)
	})

	s.Run("major version changes affect fingerprint", func() {
		ua1 := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		ua2 := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"

		fp1 := s.svc.ComputeFingerprint(ua1)
		fp2 := s.svc.ComputeFingerprint(ua2)

		s.NotEqual(fp1, fp2)
	})
}

// TestFingerprintComparison tests the drift detection logic.
func (s *DeviceServiceSuite) TestFingerprintComparison() {
	s.Run("mismatch reports drift", func() {
		matched, drift := s.svc.CompareFingerprints("a", "b")
		s.False(matched)
		s.True(drift)
	})

	s.Run("match reports no drift", func() {
		matched, drift := s.svc.CompareFingerprints("abc", "abc")
		s.True(matched)
		s.False(drift)
	})
}
