package domain

// DecisionOutcome enumerates the possible gateway decisions.
type DecisionOutcome string

const (
	DecisionPass               DecisionOutcome = "pass"
	DecisionPassWithConditions DecisionOutcome = "pass_with_conditions"
	DecisionFail               DecisionOutcome = "fail"
)

// DerivedIdentity holds non-PII attributes used in decision making.
type DerivedIdentity struct {
	// Derived fields only; no raw PII.
	PseudonymousID string
	IsOver18       bool
	CitizenValid   bool
}

// DecisionInput groups the signals considered by the decision engine. It avoids
// raw PII by requiring derived identity attributes.
type DecisionInput struct {
	Identity   DerivedIdentity
	Sanctions  SanctionsRecord
	Credential VCClaims
}
