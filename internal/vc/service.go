package vc

import (
	"context"

	"id-gateway/internal/domain"
)

// Service coordinates VC lifecycle operations while keeping signing/verification
// details encapsulated in the domain VCLifecycle.
type Service struct {
	lifecycle *domain.VCLifecycle
}

func NewService(lifecycle *domain.VCLifecycle) *Service {
	return &Service{lifecycle: lifecycle}
}

func (s *Service) Issue(ctx context.Context, req domain.IssueVCRequest) (domain.IssueVCResult, error) {
	_ = ctx
	return s.lifecycle.Issue(req)
}

func (s *Service) Verify(ctx context.Context, req domain.VerifyVCRequest) (domain.VerifyVCResult, error) {
	_ = ctx
	return s.lifecycle.Verify(req)
}
