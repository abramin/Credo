package ports

import (
	"context"

	"credo/pkg/platform/audit"
)

// AuditPort defines the interface for emitting audit events.
// This matches the audit.Emitter interface but is defined here
// to maintain hexagonal boundaries.
type AuditPort interface {
	Emit(ctx context.Context, event audit.Event) error
}
