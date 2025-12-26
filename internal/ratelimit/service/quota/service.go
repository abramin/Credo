package quota

import (
	"context"
	"fmt"
	"log/slog"

	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/ports"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
)

// Type aliases for shared interfaces.
type (
	Store          = ports.QuotaStore
	AuditPublisher = ports.AuditPublisher
)

type Service struct {
	store          Store
	auditPublisher AuditPublisher
	logger         *slog.Logger
}

type Option func(*Service)

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

func WithAuditPublisher(publisher AuditPublisher) Option {
	return func(s *Service) {
		s.auditPublisher = publisher
	}
}

func New(store Store, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("quota store is required")
	}

	svc := &Service{
		store: store,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

func (s *Service) Check(ctx context.Context, apiKeyID id.APIKeyID) (*models.APIKeyQuota, error) {
	quota, err := s.store.GetQuota(ctx, apiKeyID)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to get API key quota")
	}
	if quota == nil {
		return nil, dErrors.Wrap(fmt.Errorf("quota not found for API key %s", apiKeyID), dErrors.CodeNotFound, "quota not found")
	}
	return quota, nil
}

func (s *Service) Increment(ctx context.Context, apiKeyID id.APIKeyID, count int) (*models.APIKeyQuota, error) {
	quota, err := s.store.IncrementUsage(ctx, apiKeyID, count)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to increment API key usage")
	}

	// Log if quota exceeded
	if quota != nil && quota.CurrentUsage > quota.MonthlyLimit && quota.MonthlyLimit > 0 {
		ports.LogAudit(ctx, s.logger, s.auditPublisher, "api_key_quota_exceeded",
			"api_key_id", apiKeyID,
			"current_usage", quota.CurrentUsage,
			"monthly_limit", quota.MonthlyLimit,
		)
	}

	return quota, nil
}

func (s *Service) Reset(ctx context.Context, apiKeyID id.APIKeyID) error {
	if apiKeyID.IsNil() {
		return dErrors.New(dErrors.CodeBadRequest, "api_key_id is required")
	}

	if err := s.store.ResetQuota(ctx, apiKeyID); err != nil {
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to reset quota")
	}

	ports.LogAudit(ctx, s.logger, s.auditPublisher, "api_key_quota_reset",
		"api_key_id", apiKeyID,
	)

	return nil
}

func (s *Service) List(ctx context.Context) ([]*models.APIKeyQuota, error) {
	quotas, err := s.store.ListQuotas(ctx)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to list quotas")
	}
	return quotas, nil
}

func (s *Service) UpdateTier(ctx context.Context, apiKeyID id.APIKeyID, tier models.QuotaTier) error {
	if apiKeyID.IsNil() {
		return dErrors.New(dErrors.CodeBadRequest, "api_key_id is required")
	}
	if !tier.IsValid() {
		return dErrors.New(dErrors.CodeBadRequest, "invalid tier")
	}

	if err := s.store.UpdateTier(ctx, apiKeyID, tier); err != nil {
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to update tier")
	}

	ports.LogAudit(ctx, s.logger, s.auditPublisher, "api_key_tier_updated",
		"api_key_id", apiKeyID,
		"tier", tier,
	)

	return nil
}
