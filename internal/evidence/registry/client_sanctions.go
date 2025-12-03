package registry

import (
	"context"
	"time"
)

// SanctionsClient queries a sanctions list. The gateway keeps the interface
// small so tests can stub quickly.
type SanctionsClient interface {
	Check(ctx context.Context, nationalID string) (SanctionsRecord, error)
}

type MockSanctionsClient struct {
	Latency time.Duration
	Listed  bool
}

func (c MockSanctionsClient) Check(_ context.Context, nationalID string) (SanctionsRecord, error) {
	time.Sleep(c.Latency)
	return SanctionsRecord{
		NationalID: nationalID,
		Listed:     c.Listed,
		Source:     "mock_sanctions",
	}, nil
}
