package domain

import dErrors "credo/pkg/domain-errors"

// ConsentPurpose labels why data is processed. Purpose binding allows selective
// revocation without affecting other flows.
type ConsentPurpose string

// Supported consent purposes.
// These should align with the purposes defined in the PRD and system design.
const (
	ConsentPurposeLogin         ConsentPurpose = "login"
	ConsentPurposeRegistryCheck ConsentPurpose = "registry_check"
	ConsentPurposeVCIssuance    ConsentPurpose = "vc_issuance"
	ConsentPurposeDecision      ConsentPurpose = "decision_evaluation"
)

// validConsentPurposes is the single source of truth for valid consent purposes.
var validConsentPurposes = map[ConsentPurpose]bool{
	ConsentPurposeLogin:         true,
	ConsentPurposeRegistryCheck: true,
	ConsentPurposeVCIssuance:    true,
	ConsentPurposeDecision:      true,
}

// ParseConsentPurpose creates a ConsentPurpose from a string, validating it against
// the allowed set. Returns error if the purpose is empty or unsupported.
func ParseConsentPurpose(s string) (ConsentPurpose, error) {
	if s == "" {
		return "", dErrors.New(dErrors.CodeInvalidInput, "purpose cannot be empty")
	}
	p := ConsentPurpose(s)
	if !p.IsValid() {
		return "", dErrors.New(dErrors.CodeInvalidInput, "invalid purpose")
	}
	return p, nil
}

// IsValid checks if the consent purpose is one of the supported enum values.
func (p ConsentPurpose) IsValid() bool {
	return validConsentPurposes[p]
}

// String returns the string representation of the purpose.
func (p ConsentPurpose) String() string {
	return string(p)
}
