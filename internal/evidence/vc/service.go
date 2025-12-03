package vc

import "context"

// Service coordinates VC lifecycle operations while keeping signing/verification
// details encapsulated in the VCLifecycle.
type Service struct {
	lifecycle *VCLifecycle
	store     Store
}

func NewService(lifecycle *VCLifecycle, store Store) *Service {
	return &Service{lifecycle: lifecycle, store: store}
}

func (s *Service) Issue(ctx context.Context, req IssueRequest) (IssueResult, error) {
	_ = ctx
	result, err := s.lifecycle.Issue(req)
	if err != nil {
		return IssueResult{}, err
	}
	if s.store != nil {
		_ = s.store.Save(ctx, result)
	}
	return result, nil
}

func (s *Service) Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	_ = ctx
	return s.lifecycle.Verify(req)
}
