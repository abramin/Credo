package decision

import "id-gateway/internal/domain"

// Engine evaluates identity, registry outputs, and VC claims to produce a final
// decision. The goal is to keep the rules centralized and testable.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Evaluate(input domain.DecisionInput) domain.DecisionOutcome {
	if input.Sanctions.Listed {
		return domain.DecisionFail
	}
	if input.Identity.CitizenValid && input.Identity.IsOver18 && len(input.Credential) > 0 {
		return domain.DecisionPass
	}
	return domain.DecisionPassWithConditions
}
