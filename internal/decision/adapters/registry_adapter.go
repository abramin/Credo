package adapters

import (
	"context"

	"credo/internal/decision/ports"
	registryService "credo/internal/evidence/registry/service"
	id "credo/pkg/domain"
)

// RegistryAdapter is an in-process adapter that implements ports.RegistryPort
// by directly calling the registry service. This maintains the hexagonal
// architecture boundaries while keeping everything in a single process.
// When splitting into microservices, this can be replaced with a gRPC adapter
// without changing the decision domain layer.
type RegistryAdapter struct {
	registryService *registryService.Service
}

// NewRegistryAdapter creates a new in-process registry adapter
func NewRegistryAdapter(registryService *registryService.Service) ports.RegistryPort {
	return &RegistryAdapter{
		registryService: registryService,
	}
}

// CheckCitizen retrieves citizen record by national ID
// Uses CitizenWithDetails to get DOB for age derivation (decision service will minimize its own output)
func (a *RegistryAdapter) CheckCitizen(ctx context.Context, userID id.UserID, nationalID id.NationalID) (*ports.CitizenRecord, error) {
	record, err := a.registryService.CitizenWithDetails(ctx, userID, nationalID)
	if err != nil {
		return nil, err
	}

	return &ports.CitizenRecord{
		NationalID:  record.NationalID,
		FullName:    record.FullName,
		DateOfBirth: record.DateOfBirth,
		Valid:       record.Valid,
		CheckedAt:   record.CheckedAt,
	}, nil
}

// CheckSanctions retrieves sanctions record by national ID
func (a *RegistryAdapter) CheckSanctions(ctx context.Context, userID id.UserID, nationalID id.NationalID) (*ports.SanctionsRecord, error) {
	record, err := a.registryService.Sanctions(ctx, userID, nationalID)
	if err != nil {
		return nil, err
	}

	return &ports.SanctionsRecord{
		NationalID: record.NationalID,
		Listed:     record.Listed,
		Source:     record.Source,
		CheckedAt:  record.CheckedAt,
	}, nil
}

// Check performs combined citizen + sanctions lookup
// Uses CitizenWithDetails for citizen to get DOB for age derivation
func (a *RegistryAdapter) Check(ctx context.Context, userID id.UserID, nationalID id.NationalID) (*ports.CitizenRecord, *ports.SanctionsRecord, error) {
	// Use CitizenWithDetails to bypass regulated mode minimization
	// The decision service needs DOB to compute is_over_18 and will minimize its own output
	citizenRecord, err := a.registryService.CitizenWithDetails(ctx, userID, nationalID)
	if err != nil {
		return nil, nil, err
	}

	sanctionsRecord, err := a.registryService.Sanctions(ctx, userID, nationalID)
	if err != nil {
		return nil, nil, err
	}

	citizen := &ports.CitizenRecord{
		NationalID:  citizenRecord.NationalID,
		FullName:    citizenRecord.FullName,
		DateOfBirth: citizenRecord.DateOfBirth,
		Valid:       citizenRecord.Valid,
		CheckedAt:   citizenRecord.CheckedAt,
	}

	sanctions := &ports.SanctionsRecord{
		NationalID: sanctionsRecord.NationalID,
		Listed:     sanctionsRecord.Listed,
		Source:     sanctionsRecord.Source,
		CheckedAt:  sanctionsRecord.CheckedAt,
	}

	return citizen, sanctions, nil
}
