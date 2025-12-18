package models

import "time"

// RateLimitExceededResponse is the API response when rate limit is exceeded.
type RateLimitExceededResponse struct {
	Error      string `json:"error"` // "rate_limit_exceeded" or "user_rate_limit_exceeded"
	Message    string `json:"message"`
	RetryAfter int    `json:"retry_after"` // seconds
}

// UserRateLimitExceededResponse is the API response when user quota is exceeded.
type UserRateLimitExceededResponse struct {
	Error          string    `json:"error"` // "user_rate_limit_exceeded"
	Message        string    `json:"message"`
	QuotaLimit     int       `json:"quota_limit"`
	QuotaRemaining int       `json:"quota_remaining"`
	QuotaReset     time.Time `json:"quota_reset"`
}

// AllowlistEntryResponse is the API response for allowlist operations.
type AllowlistEntryResponse struct {
	Allowlisted bool       `json:"allowlisted"`
	Identifier  string     `json:"identifier"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// QuotaResponse is the API response with quota headers info.
type QuotaResponse struct {
	QuotaLimit     int       `json:"quota_limit"`
	QuotaRemaining int       `json:"quota_remaining"`
	QuotaReset     time.Time `json:"quota_reset"`
}

// ServiceOverloadedResponse is the API response when global throttle is hit.
type ServiceOverloadedResponse struct {
	Error      string `json:"error"`   // "service_unavailable"
	Message    string `json:"message"` // "Service is temporarily overloaded..."
	RetryAfter int    `json:"retry_after"`
}
