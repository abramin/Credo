package handler

import (
	"strings"

	"credo/internal/decision"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
)

// EvaluateRequest is the HTTP request body for POST /decision/evaluate.
type EvaluateRequest struct {
	Purpose string         `json:"purpose"`
	Context RequestContext `json:"context"`

	// Parsed values (populated by Validate)
	parsedPurpose    decision.Purpose
	parsedNationalID id.NationalID
}

// RequestContext holds context-specific data for evaluation.
type RequestContext struct {
	NationalID string `json:"national_id"`
}

// Validate validates and parses the request.
// Implements the Validatable interface for httputil.DecodeAndPrepare.
func (r *EvaluateRequest) Validate() error {
	if r == nil {
		return dErrors.New(dErrors.CodeBadRequest, "request body is required")
	}

	// Size validation (fail fast)
	if len(r.Context.NationalID) > 20 {
		return dErrors.New(dErrors.CodeValidation, "national_id must be at most 20 characters")
	}

	// Required fields
	r.Purpose = strings.TrimSpace(r.Purpose)
	if r.Purpose == "" {
		return dErrors.New(dErrors.CodeValidation, "purpose is required")
	}

	// Parse purpose
	purpose, err := decision.ParsePurpose(r.Purpose)
	if err != nil {
		return err
	}
	r.parsedPurpose = purpose

	// Purpose-specific validation
	r.Context.NationalID = strings.TrimSpace(r.Context.NationalID)
	switch purpose {
	case decision.PurposeAgeVerification, decision.PurposeSanctionsScreening:
		if r.Context.NationalID == "" {
			return dErrors.New(dErrors.CodeValidation, "context.national_id is required")
		}
		nationalID, err := id.ParseNationalID(r.Context.NationalID)
		if err != nil {
			return err
		}
		r.parsedNationalID = nationalID
	}

	return nil
}

// ParsedPurpose returns the validated purpose.
func (r *EvaluateRequest) ParsedPurpose() decision.Purpose {
	return r.parsedPurpose
}

// ParsedNationalID returns the validated national ID.
func (r *EvaluateRequest) ParsedNationalID() id.NationalID {
	return r.parsedNationalID
}
