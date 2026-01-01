package handler

import (
	"time"

	"credo/internal/decision"
)

// EvaluateResponse is the HTTP response for POST /decision/evaluate.
type EvaluateResponse struct {
	Status      string           `json:"status"`
	Reason      string           `json:"reason"`
	Conditions  []string         `json:"conditions"`
	Evidence    EvidenceResponse `json:"evidence"`
	EvaluatedAt time.Time        `json:"evaluated_at"`
}

// EvidenceResponse is the evidence portion of the response.
type EvidenceResponse struct {
	CitizenValid    *bool `json:"citizen_valid,omitempty"`
	SanctionsListed bool  `json:"sanctions_listed"`
	HasCredential   *bool `json:"has_credential,omitempty"`
	IsOver18        *bool `json:"is_over_18,omitempty"`
}

// FromResult converts a domain EvaluateResult to an HTTP response.
func FromResult(result *decision.EvaluateResult) *EvaluateResponse {
	return &EvaluateResponse{
		Status:     string(result.Status),
		Reason:     string(result.Reason),
		Conditions: result.Conditions,
		Evidence: EvidenceResponse{
			CitizenValid:    result.Evidence.CitizenValid,
			SanctionsListed: result.Evidence.SanctionsListed,
			HasCredential:   result.Evidence.HasCredential,
			IsOver18:        result.Evidence.IsOver18,
		},
		EvaluatedAt: result.EvaluatedAt,
	}
}
