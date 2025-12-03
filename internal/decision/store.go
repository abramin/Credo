package decision

import "context"

// Store persists decision outcomes for auditability. Swap with concrete storage
// without touching the service.
type Store interface {
	Save(ctx context.Context, input DecisionInput, outcome DecisionOutcome) error
}
