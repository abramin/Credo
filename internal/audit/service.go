package audit

import (
	"context"
	"time"

	"id-gateway/internal/domain"
	"id-gateway/internal/storage"
)

// Service captures structured audit events. It is append-only and uses the
// storage layer for persistence so tests can swap sinks easily.
type Service struct {
	store storage.AuditStore
}

func NewService(store storage.AuditStore) *Service {
	return &Service{store: store}
}

func (s *Service) Emit(ctx context.Context, base domain.AuditEvent) error {
	if base.Timestamp.IsZero() {
		base.Timestamp = time.Now()
	}
	return s.store.Append(ctx, base)
}

func (s *Service) List(ctx context.Context, userID string) ([]domain.AuditEvent, error) {
	return s.store.ListByUser(ctx, userID)
}
