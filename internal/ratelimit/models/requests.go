package models

import (
	"strings"
	"time"

	dErrors "credo/pkg/domain-errors"
)

// AddAllowlistRequest is the API request for adding an allowlist entry.
type AddAllowlistRequest struct {
	Type       string     `json:"type"`       // "ip" or "user_id"
	Identifier string     `json:"identifier"`
	Reason     string     `json:"reason"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// Normalize trims and sanitizes input fields.
func (r *AddAllowlistRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = strings.TrimSpace(strings.ToLower(r.Type))
	r.Identifier = strings.TrimSpace(r.Identifier)
	r.Reason = strings.TrimSpace(r.Reason)
}

// Validate validates the AddAllowlistRequest fields.
// Follows validation order: Size -> Required -> Syntax -> Semantic.
func (r *AddAllowlistRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	// Phase 1: Size checks
	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}
	if len(r.Reason) > 500 {
		return dErrors.New(dErrors.CodeValidation, "reason must be 500 characters or less")
	}

	// Phase 2: Required fields
	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}
	if r.Reason == "" {
		return dErrors.New(dErrors.CodeValidation, "reason is required")
	}

	// Phase 3: Syntax validation
	entryType := AllowlistEntryType(r.Type)
	if !entryType.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	// Phase 4: Semantic validation
	if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) {
		return dErrors.New(dErrors.CodeValidation, "expires_at must be in the future")
	}

	return nil
}

// RemoveAllowlistRequest is the API request for removing an allowlist entry.
type RemoveAllowlistRequest struct {
	Type       string `json:"type"`       // "ip" or "user_id"
	Identifier string `json:"identifier"`
}

// Normalize trims and sanitizes input fields.
func (r *RemoveAllowlistRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = strings.TrimSpace(strings.ToLower(r.Type))
	r.Identifier = strings.TrimSpace(r.Identifier)
}

// Validate validates the RemoveAllowlistRequest fields.
func (r *RemoveAllowlistRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	// Phase 1: Size checks
	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}

	// Phase 2: Required fields
	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}

	// Phase 3: Syntax validation
	entryType := AllowlistEntryType(r.Type)
	if !entryType.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	return nil
}

// ResetRateLimitRequest is the API request for resetting a rate limit counter.
// Admin operation per PRD-017 TR-1.
type ResetRateLimitRequest struct {
	Type       string `json:"type"`            // "ip" or "user_id"
	Identifier string `json:"identifier"`
	Class      string `json:"class,omitempty"` // optional: specific endpoint class to reset
}

// Normalize trims and sanitizes input fields.
func (r *ResetRateLimitRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = strings.TrimSpace(strings.ToLower(r.Type))
	r.Identifier = strings.TrimSpace(r.Identifier)
	r.Class = strings.TrimSpace(strings.ToLower(r.Class))
}

// Validate validates the ResetRateLimitRequest fields.
func (r *ResetRateLimitRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	// Phase 1: Size checks
	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}

	// Phase 2: Required fields
	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}

	// Phase 3: Syntax validation
	entryType := AllowlistEntryType(r.Type)
	if !entryType.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	// Validate class if provided
	if r.Class != "" {
		class := EndpointClass(r.Class)
		if !class.IsValid() {
			return dErrors.New(dErrors.CodeValidation, "class must be 'auth', 'sensitive', 'read', or 'write'")
		}
	}

	return nil
}
