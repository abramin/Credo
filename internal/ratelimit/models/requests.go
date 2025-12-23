package models

import (
	"strings"
	"time"

	dErrors "credo/pkg/domain-errors"
)

type AddAllowlistRequest struct {
	Type       AllowlistEntryType `json:"type"`
	Identifier string             `json:"identifier"`
	Reason     string             `json:"reason"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
}

func (r *AddAllowlistRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = AllowlistEntryType(strings.TrimSpace(strings.ToLower(string(r.Type))))
	r.Identifier = strings.TrimSpace(r.Identifier)
	r.Reason = strings.TrimSpace(r.Reason)
}

// Follows validation order: Size -> Required -> Syntax -> Semantic.
func (r *AddAllowlistRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}
	if len(r.Reason) > 500 {
		return dErrors.New(dErrors.CodeValidation, "reason must be 500 characters or less")
	}

	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}
	if r.Reason == "" {
		return dErrors.New(dErrors.CodeValidation, "reason is required")
	}

	entryType := AllowlistEntryType(r.Type)
	if !entryType.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) {
		return dErrors.New(dErrors.CodeValidation, "expires_at must be in the future")
	}

	return nil
}

type RemoveAllowlistRequest struct {
	Type       AllowlistEntryType `json:"type"`
	Identifier string             `json:"identifier"`
}

func (r *RemoveAllowlistRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = AllowlistEntryType(strings.TrimSpace(strings.ToLower(string(r.Type))))
	r.Identifier = strings.TrimSpace(r.Identifier)
}

// Follows validation order: Size -> Required -> Syntax -> Semantic.
func (r *RemoveAllowlistRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}

	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}

	entryType := AllowlistEntryType(r.Type)
	if !entryType.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	return nil
}

type ResetRateLimitRequest struct {
	Type       AllowlistEntryType `json:"type"`
	Identifier string             `json:"identifier"`
	Class      EndpointClass      `json:"class,omitempty"` // optional: specific endpoint class to reset
}

func (r *ResetRateLimitRequest) Normalize() {
	if r == nil {
		return
	}
	r.Type = AllowlistEntryType(strings.TrimSpace(strings.ToLower(string(r.Type))))
	r.Identifier = strings.TrimSpace(r.Identifier)
	r.Class = EndpointClass(strings.TrimSpace(strings.ToLower(string(r.Class))))
}

// Follows validation order: Size -> Required -> Syntax -> Semantic.
func (r *ResetRateLimitRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request is required")
	}

	if len(r.Identifier) > 255 {
		return dErrors.New(dErrors.CodeValidation, "identifier must be 255 characters or less")
	}

	if r.Type == "" {
		return dErrors.New(dErrors.CodeValidation, "type is required")
	}
	if r.Identifier == "" {
		return dErrors.New(dErrors.CodeValidation, "identifier is required")
	}

	if !r.Type.IsValid() {
		return dErrors.New(dErrors.CodeValidation, "type must be 'ip' or 'user_id'")
	}

	if r.Class != "" {
		if !r.Class.IsValid() {
			return dErrors.New(dErrors.CodeValidation, "class must be 'auth', 'sensitive', 'read', or 'write'")
		}
	}

	return nil
}
