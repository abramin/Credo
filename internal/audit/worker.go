package audit

import "context"

// Worker consumes audit events from a channel and persists them. It keeps
// background processing testable without wiring queue implementations yet.
type Worker struct {
	store Store
	inbox <-chan Event
}

func NewWorker(store Store, inbox <-chan Event) *Worker {
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
