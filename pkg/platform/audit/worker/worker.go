package worker

import (
	"context"

	audit "credo/pkg/platform/audit"
)

// Worker consumes audit events from a channel and persists them. It keeps
// background processing testable without wiring queue implementations yet.
type Worker struct {
	store audit.Store
	inbox <-chan audit.Event
}

func NewWorker(store audit.Store, inbox <-chan audit.Event) *Worker {
	return &Worker{store: store, inbox: inbox}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-w.inbox:
			if err := w.store.Append(ctx, event); err != nil {
				return err
			}
		}
	}
}
