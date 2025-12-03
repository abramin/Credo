package registry

import (
	"context"
	"time"
)

// CitizenClient queries a citizen registry. Mock implementations use
// deterministic data and a configurable latency to mimic real-world calls.
type CitizenClient interface {
	Lookup(ctx context.Context, nationalID string) (CitizenRecord, error)
}

type MockCitizenClient struct {
	Latency       time.Duration
	RegulatedMode bool
}

func (c MockCitizenClient) Lookup(_ context.Context, nationalID string) (CitizenRecord, error) {
	time.Sleep(c.Latency)
	record := CitizenRecord{
		NationalID:  nationalID,
		FullName:    "Sample Citizen",
		DateOfBirth: "1990-02-03",
		Valid:       true,
	}
	if c.RegulatedMode {
		return MinimizeCitizenRecord(record), nil
	}
	return record, nil
}
