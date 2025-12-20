package audit

import (
	"context"
	"time"

	id "credo/pkg/domain"
)

// Publisher captures structured audit events. It is append-only and uses the
// storage layer for persistence so tests can swap sinks easily.
type Publisher struct {
	store Store
}

func NewPublisher(store Store) *Publisher {
	return &Publisher{store: store}
}

func (p *Publisher) Emit(ctx context.Context, base Event) error {
	if base.Timestamp.IsZero() {
		base.Timestamp = time.Now()
	}
	return p.store.Append(ctx, base)
}

func (p *Publisher) List(ctx context.Context, userID id.UserID) ([]Event, error) {
	return p.store.ListByUser(ctx, userID)
}
