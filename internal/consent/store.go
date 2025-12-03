package consent

import (
	"context"
	"time"
)

type Store interface {
	Save(ctx context.Context, consent ConsentRecord) error
	ListByUser(ctx context.Context, userID string) ([]ConsentRecord, error)
	Revoke(ctx context.Context, userID string, purpose ConsentPurpose, revokedAt time.Time) error
}
