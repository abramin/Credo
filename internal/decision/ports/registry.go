package ports

import (
	"context"
	"time"
)

// RegistryPort defines the interface for registry lookups
// This port allows the decision engine to fetch identity evidence
// without depending on gRPC, HTTP, or specific registry implementations.
type RegistryPort interface {
	// CheckCitizen retrieves citizen record by national ID
	// Returns minimized data in regulated mode
	CheckCitizen(ctx context.Context, nationalID string) (*CitizenRecord, error)

	// CheckSanctions retrieves sanctions record by national ID
	// Always returns minimal data (no PII)
	CheckSanctions(ctx context.Context, nationalID string) (*SanctionsRecord, error)

	// Check performs combined citizen + sanctions lookup
	// Optimized for parallel execution
	Check(ctx context.Context, nationalID string) (*CitizenRecord, *SanctionsRecord, error)
}

// CitizenRecord represents citizen identity data (port model)
// This is a domain model, not a protobuf or database model
type CitizenRecord struct {
	NationalID  string
	FullName    string // Empty in regulated mode
	DateOfBirth string // Empty in regulated mode (YYYY-MM-DD)
	Valid       bool
	CheckedAt   time.Time
}

// SanctionsRecord represents sanctions/PEP status (port model)
// No PII - safe for any mode
type SanctionsRecord struct {
	NationalID string
	Listed     bool
	Source     string
	CheckedAt  time.Time
}
