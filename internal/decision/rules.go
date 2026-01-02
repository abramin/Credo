package decision

import "time"

// EvaluateDecision applies purpose-specific rules to produce an outcome.
// This is pure domain logic - no I/O, no side effects.
// The function receives all data it needs as arguments and returns a decision.
func EvaluateDecision(purpose Purpose, input DecisionInput) DecisionOutcome {
	switch purpose {
	case PurposeAgeVerification:
		return evaluateAgeVerification(input)
	case PurposeSanctionsScreening:
		return evaluateSanctionsScreening(input)
	default:
		return DecisionFail
	}
}

// evaluateAgeVerification applies the age verification rule chain.
// Rule priority (fail-fast):
//  1. Sanctions check (hard fail) - compliance-critical
//  2. Citizen validity - identity baseline
//  3. Age requirement - purpose-specific
//  4. Credential check (soft requirement for full pass)
func evaluateAgeVerification(input DecisionInput) DecisionOutcome {
	// Rule 1: Sanctions check (hard fail) - compliance-critical
	if input.IsSanctioned() {
		return DecisionFail
	}

	// Rule 2: Citizen validity - identity baseline
	if !input.IsCitizenValid() {
		return DecisionFail
	}

	// Rule 3: Age requirement - purpose-specific
	if !input.IsOfLegalAge() {
		return DecisionFail
	}

	// Rule 4: Credential check (soft requirement for full pass)
	if len(input.Credential) > 0 {
		return DecisionPass
	}

	return DecisionPassWithConditions
}

// evaluateSanctionsScreening applies sanctions-only screening rules.
func evaluateSanctionsScreening(input DecisionInput) DecisionOutcome {
	if input.IsSanctioned() {
		return DecisionFail
	}
	return DecisionPass
}

// isSanctionsDecision returns true if the decision involves sanctions
// (either by purpose or by result reason).
func isSanctionsDecision(req EvaluateRequest, result *EvaluateResult) bool {
	return result.Reason == ReasonSanctioned || req.Purpose == PurposeSanctionsScreening
}

// BuildResult constructs an EvaluateResult from the evaluation outcome.
// This is pure domain logic - no I/O, no side effects.
func BuildResult(purpose Purpose, outcome DecisionOutcome, evidence *GatheredEvidence, derived DerivedIdentity, evalTime time.Time) *EvaluateResult {
	result := &EvaluateResult{
		Status:      outcome,
		EvaluatedAt: evalTime,
		Conditions:  []string{},
		Evidence: EvidenceSummary{
			SanctionsListed: evidence.Sanctions != nil && evidence.Sanctions.Listed,
		},
	}

	switch purpose {
	case PurposeAgeVerification:
		return buildAgeVerificationResult(result, outcome, evidence, derived)
	case PurposeSanctionsScreening:
		return buildSanctionsResult(result, outcome, evidence)
	}

	return result
}

func buildAgeVerificationResult(result *EvaluateResult, outcome DecisionOutcome, evidence *GatheredEvidence, derived DerivedIdentity) *EvaluateResult {
	// Set evidence fields
	if evidence.Citizen != nil {
		valid := evidence.Citizen.Valid
		result.Evidence.CitizenValid = &valid
	}
	over18 := derived.IsOver18
	result.Evidence.IsOver18 = &over18
	hasCred := evidence.Credential != nil
	result.Evidence.HasCredential = &hasCred

	// Set reason based on failure point
	switch outcome {
	case DecisionFail:
		if evidence.Sanctions != nil && evidence.Sanctions.Listed {
			result.Reason = ReasonSanctioned
		} else if evidence.Citizen == nil || !evidence.Citizen.Valid {
			result.Reason = ReasonInvalidCitizen
		} else if !derived.IsOver18 {
			result.Reason = ReasonUnderage
		}
	case DecisionPass:
		result.Reason = ReasonAllChecksPassed
	case DecisionPassWithConditions:
		result.Reason = ReasonMissingCredential
		result.Conditions = []string{"obtain_age_credential"}
	}

	return result
}

func buildSanctionsResult(result *EvaluateResult, outcome DecisionOutcome, evidence *GatheredEvidence) *EvaluateResult {
	switch outcome {
	case DecisionFail:
		result.Reason = ReasonSanctioned
	case DecisionPass:
		result.Reason = ReasonNotSanctioned
	}
	return result
}
