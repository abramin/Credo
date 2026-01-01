package decision

import (
	"context"
	"time"

	vcmodels "credo/internal/evidence/vc/models"

	"golang.org/x/sync/errgroup"
)

// gatherEvidence orchestrates parallel evidence gathering with shared context cancellation.
func (s *Service) gatherEvidence(ctx context.Context, req EvaluateRequest) (*GatheredEvidence, error) {
	ctx, cancel := context.WithTimeout(ctx, evidenceTimeout)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	evidence := &GatheredEvidence{
		FetchedAt: time.Now(),
	}

	// Launch evidence fetches based on purpose
	switch req.Purpose {
	case PurposeAgeVerification:
		s.gatherAgeVerificationEvidence(ctx, g, evidence, req)
	case PurposeSanctionsScreening:
		s.gatherSanctionsEvidence(ctx, g, evidence, req)
	}

	// Wait for all goroutines with early cancellation on first failure
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return evidence, nil
}

func (s *Service) gatherAgeVerificationEvidence(
	ctx context.Context,
	g *errgroup.Group,
	evidence *GatheredEvidence,
	req EvaluateRequest,
) {
	// Fetch citizen record
	g.Go(func() error {
		start := time.Now()
		citizen, err := s.registry.CheckCitizen(ctx, req.UserID, req.NationalID)
		evidence.Latencies.Citizen = time.Since(start)

		if s.metrics != nil {
			s.metrics.ObserveEvidenceLatency("citizen", evidence.Latencies.Citizen)
		}

		if err != nil {
			return err
		}
		evidence.Citizen = citizen
		return nil
	})

	// Fetch sanctions record
	g.Go(func() error {
		start := time.Now()
		sanctions, err := s.registry.CheckSanctions(ctx, req.UserID, req.NationalID)
		evidence.Latencies.Sanctions = time.Since(start)

		if s.metrics != nil {
			s.metrics.ObserveEvidenceLatency("sanctions", evidence.Latencies.Sanctions)
		}

		if err != nil {
			return err
		}
		evidence.Sanctions = sanctions
		return nil
	})

	// Fetch existing AgeOver18 VC (optional - no error on not found)
	g.Go(func() error {
		start := time.Now()
		cred, err := s.vc.FindBySubjectAndType(ctx, req.UserID, vcmodels.CredentialTypeAgeOver18)
		evidence.Latencies.Credential = time.Since(start)

		if s.metrics != nil {
			s.metrics.ObserveEvidenceLatency("credential", evidence.Latencies.Credential)
		}

		// Not finding a credential is not an error - it's just missing evidence
		if err != nil {
			if s.logger != nil {
				s.logger.DebugContext(ctx, "credential lookup failed",
					"user_id", req.UserID,
					"error", err,
				)
			}
			// Don't return error - credential is optional
			return nil
		}
		evidence.Credential = cred
		return nil
	})
}

func (s *Service) gatherSanctionsEvidence(
	ctx context.Context,
	g *errgroup.Group,
	evidence *GatheredEvidence,
	req EvaluateRequest,
) {
	// Only fetch sanctions for sanctions-only screening
	g.Go(func() error {
		start := time.Now()
		sanctions, err := s.registry.CheckSanctions(ctx, req.UserID, req.NationalID)
		evidence.Latencies.Sanctions = time.Since(start)

		if s.metrics != nil {
			s.metrics.ObserveEvidenceLatency("sanctions", evidence.Latencies.Sanctions)
		}

		if err != nil {
			return err
		}
		evidence.Sanctions = sanctions
		return nil
	})
}
