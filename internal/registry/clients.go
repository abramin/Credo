package registry

import (
	"context"
	"time"

	"id-gateway/internal/domain"
)

// CitizenClient queries a citizen registry. Mock implementations use deterministic
// data and a configurable latency to mimic real-world calls.
type CitizenClient interface {
	Lookup(ctx context.Context, nationalID string) (domain.CitizenRecord, error)
}

// SanctionsClient queries a sanctions list. The gateway keeps the interface small
// so tests can stub quickly.
type SanctionsClient interface {
	Check(ctx context.Context, nationalID string) (domain.SanctionsRecord, error)
}

type MockCitizenClient struct {
	Latency       time.Duration
	RegulatedMode bool
}

func (c MockCitizenClient) Lookup(_ context.Context, nationalID string) (domain.CitizenRecord, error) {
	time.Sleep(c.Latency)
	record := domain.CitizenRecord{
		NationalID:  nationalID,
		FullName:    "Sample Citizen",
		DateOfBirth: "1990-02-03",
		Valid:       true,
	}
	if c.RegulatedMode {
		return domain.MinimizeCitizenRecord(record), nil
	}
	return record, nil
}

type MockSanctionsClient struct {
	Latency time.Duration
	// Flag controls deterministic sanctioning for tests.
	Listed bool
}

func (c MockSanctionsClient) Check(_ context.Context, nationalID string) (domain.SanctionsRecord, error) {
	time.Sleep(c.Latency)
	return domain.SanctionsRecord{
		NationalID: nationalID,
		Listed:     c.Listed,
		Source:     "mock_sanctions",
	}, nil
}
