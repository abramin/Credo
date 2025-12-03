package decision

// Service evaluates identity, registry outputs, and VC claims to produce a
// final decision. The goal is to keep the rules centralized and testable.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Evaluate(input DecisionInput) DecisionOutcome {
	if input.Sanctions.Listed {
		return DecisionFail
	}
	if input.Identity.CitizenValid && input.Identity.IsOver18 && len(input.Credential) > 0 {
		return DecisionPass
	}
	return DecisionPassWithConditions
}
