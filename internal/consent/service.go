package consent

import (
	"context"
	"time"

	pkgerrors "id-gateway/pkg/domain-errors"
)

// Service persists consent decisions and provides purpose-aware checks. It keeps
// orchestration out of handlers and domain logic thin.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Grant(ctx context.Context, userID string, purpose ConsentPurpose, ttl time.Duration) (ConsentRecord, error) {
	now := time.Now()
	record := ConsentRecord{
		UserID:    userID,
		Purpose:   purpose,
		GrantedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	if err := s.store.Save(ctx, record); err != nil {
		return ConsentRecord{}, err
	}
	return record, nil
}

// Require returns an error when consent is missing or expired.
func (s *Service) Require(ctx context.Context, userID string, purpose ConsentPurpose, now time.Time) error {
	consents, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	return EnsureConsent(consents, purpose, now)
}

func (s *Service) Revoke(ctx context.Context, userID string, purpose ConsentPurpose) error {
	now := time.Now()
	if err := s.store.Revoke(ctx, userID, purpose, now); err != nil {
		return pkgerrors.New(pkgerrors.CodeInvalidConsent, err.Error())
	}
	return nil
}
