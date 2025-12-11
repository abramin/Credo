package providers

import (
	"errors"
	"fmt"
)

// ErrorCategory defines the normalized failure taxonomy
type ErrorCategory string

const (
	// ErrorTimeout indicates the provider took too long to respond
	ErrorTimeout ErrorCategory = "timeout"

	// ErrorBadData indicates the provider returned invalid/malformed data
	ErrorBadData ErrorCategory = "bad_data"

	// ErrorAuthentication indicates credential or permission issues
	ErrorAuthentication ErrorCategory = "authentication"

	// ErrorProviderOutage indicates the provider is unavailable
	ErrorProviderOutage ErrorCategory = "provider_outage"

	// ErrorContractMismatch indicates the provider API version changed
	ErrorContractMismatch ErrorCategory = "contract_mismatch"

	// ErrorNotFound indicates the requested record doesn't exist
	ErrorNotFound ErrorCategory = "not_found"

	// ErrorRateLimited indicates too many requests
	ErrorRateLimited ErrorCategory = "rate_limited"

	// ErrorInternal indicates an unexpected internal error
	ErrorInternal ErrorCategory = "internal"
)

// ProviderError wraps provider failures with normalized categorization
type ProviderError struct {
	Category   ErrorCategory
	ProviderID string
	Message    string
	Underlying error
	Retryable  bool // Whether this error is worth retrying
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("provider %s [%s]: %s: %v", e.ProviderID, e.Category, e.Message, e.Underlying)
	}
	return fmt.Sprintf("provider %s [%s]: %s", e.ProviderID, e.Category, e.Message)
}

// Unwrap supports error unwrapping
func (e *ProviderError) Unwrap() error {
	return e.Underlying
}

// NewProviderError creates a new normalized provider error
func NewProviderError(category ErrorCategory, providerID, message string, underlying error) *ProviderError {
	retryable := category == ErrorTimeout ||
		category == ErrorProviderOutage ||
		category == ErrorRateLimited

	return &ProviderError{
		Category:   category,
		ProviderID: providerID,
		Message:    message,
		Underlying: underlying,
		Retryable:  retryable,
	}
}

// IsRetryable checks if an error is worth retrying
func IsRetryable(err error) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Retryable
	}
	return false
}

// GetCategory extracts the error category from an error
func GetCategory(err error) ErrorCategory {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Category
	}
	return ErrorInternal
}

// Sentinel errors for common cases
var (
	ErrProviderNotFound     = errors.New("provider not found")
	ErrNoProvidersAvailable = errors.New("no providers available for this type")
	ErrAllProvidersFailed   = errors.New("all providers failed")
)
