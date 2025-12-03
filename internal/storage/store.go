package storage

import (
	"context"
	"time"

	"id-gateway/internal/domain"
)

// Stores are interface-driven to keep the domain logic testable and to allow
// swapping in-memory, file-based, or external persistence without rewiring
// business code.
type UserStore interface {
	Save(ctx context.Context, user domain.User) error
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type SessionStore interface {
	Save(ctx context.Context, session domain.Session) error
	FindByID(ctx context.Context, id string) (domain.Session, error)
}

type ConsentStore interface {
	Save(ctx context.Context, consent domain.ConsentRecord) error
	ListByUser(ctx context.Context, userID string) ([]domain.ConsentRecord, error)
	Revoke(ctx context.Context, userID string, purpose domain.ConsentPurpose, revokedAt time.Time) error
}

type VCStore interface {
	Save(ctx context.Context, credential domain.IssueVCResult) error
	FindByID(ctx context.Context, id string) (domain.IssueVCResult, error)
}

type RegistryCacheStore interface {
	SaveCitizen(ctx context.Context, record domain.CitizenRecord) error
	FindCitizen(ctx context.Context, nationalID string) (domain.CitizenRecord, error)
	SaveSanction(ctx context.Context, record domain.SanctionsRecord) error
	FindSanction(ctx context.Context, nationalID string) (domain.SanctionsRecord, error)
}

type AuditStore interface {
	Append(ctx context.Context, event domain.AuditEvent) error
	ListByUser(ctx context.Context, userID string) ([]domain.AuditEvent, error)
}
