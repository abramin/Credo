package consent

import (
	"context"
	"time"

	"id-gateway/internal/domain"
	"id-gateway/internal/storage"
	pkgerrors "id-gateway/pkg/errors"
)

// Service persists consent decisions and provides purpose-aware checks. It keeps
// orchestration out of handlers and domain logic thin.
type Service struct {
	store storage.ConsentStore
}

func NewService(store storage.ConsentStore) *Service {
	return &Service{store: store}
}

func (s *Service) Grant(ctx context.Context, userID string, purpose domain.ConsentPurpose, ttl time.Duration) (domain.ConsentRecord, error) {
	now := time.Now()
	record := domain.ConsentRecord{
		UserID:    userID,
		Purpose:   purpose,
		GrantedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	if err := s.store.Save(ctx, record); err != nil {
		return domain.ConsentRecord{}, err
	}
	return record, nil
}

// Require returns an error when consent is missing or expired.
func (s *Service) Require(ctx context.Context, userID string, purpose domain.ConsentPurpose, now time.Time) error {
	consents, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	return domain.EnsureConsent(consents, purpose, now)
}

func (s *Service) Revoke(ctx context.Context, userID string, purpose domain.ConsentPurpose) error {
	now := time.Now()
	if err := s.store.Revoke(ctx, userID, purpose, now); err != nil {
		return pkgerrors.New(pkgerrors.CodeInvalidConsent, err.Error())
	}
	return nil
}
