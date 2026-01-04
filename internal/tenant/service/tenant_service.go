package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	tenantmetrics "credo/internal/tenant/metrics"
	"credo/internal/tenant/models"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/sentinel"
	"credo/pkg/requestcontext"
)

// TenantService orchestrates tenant lifecycle management.
type TenantService struct {
	tenants      TenantStore
	auditEmitter *auditEmitter
	metrics      *tenantmetrics.Metrics
	tx           StoreTx
}

func NewTenantService(tenants TenantStore, opts ...Option) *TenantService {
	cfg := &serviceConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	tx := cfg.tx
	if tx == nil {
		tx = newInMemoryStoreTx()
	}
	return &TenantService{
		tenants:      tenants,
		auditEmitter: newAuditEmitter(cfg.logger, cfg.auditPublisher),
		metrics:      cfg.metrics,
		tx:           tx,
	}
}

func (s *TenantService) CreateTenant(ctx context.Context, name string) (*models.Tenant, error) {
	name = strings.TrimSpace(name)

	var tenant *models.Tenant
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		t, err := models.NewTenant(id.TenantID(uuid.New()), name, requestcontext.Now(txCtx))
		if err != nil {
			return err
		}

		if err := s.tenants.CreateIfNameAvailable(txCtx, t); err != nil {
			if errors.Is(err, sentinel.ErrAlreadyUsed) || dErrors.HasCode(err, dErrors.CodeConflict) {
				return dErrors.New(dErrors.CodeConflict, "tenant name must be unique")
			}
			return dErrors.Wrap(err, dErrors.CodeInternal, "failed to create tenant")
		}
		if err := s.auditEmitter.emitTenantCreated(txCtx, models.TenantCreated{TenantID: t.ID}); err != nil {
			return err
		}
		tenant = t
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.incrementTenantCreated()
	return tenant, nil
}

func (s *TenantService) GetTenant(ctx context.Context, tenantID id.TenantID) (*models.Tenant, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	tenant, err := s.tenants.FindByID(ctx, tenantID)
	if err != nil {
		return nil, wrapTenantErr(err)
	}
	return tenant, nil
}

// GetTenantByName retrieves a tenant by name (case-insensitive).
func (s *TenantService) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, dErrors.New(dErrors.CodeBadRequest, "tenant name is required")
	}
	tenant, err := s.tenants.FindByName(ctx, name)
	if err != nil {
		return nil, wrapTenantErr(err)
	}
	return tenant, nil
}

// DeactivateTenant transitions a tenant to inactive status.
// Returns the updated tenant or an error if tenant is not found or already inactive.
//
// Uses the Execute callback pattern for atomic validate-then-mutate.
// The store's Execute method holds the lock (mutex or FOR UPDATE) during both validation and mutation.
func (s *TenantService) DeactivateTenant(ctx context.Context, tenantID id.TenantID) (*models.Tenant, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}

	now := requestcontext.Now(ctx)
	tenant, err := s.tenants.Execute(ctx, tenantID,
		func(t *models.Tenant) error {
			if err := t.CanDeactivate(); err != nil {
				if dErrors.HasCode(err, dErrors.CodeInvariantViolation) {
					return dErrors.New(dErrors.CodeConflict, "tenant is already inactive")
				}
				return err
			}
			return nil
		},
		func(t *models.Tenant) {
			t.ApplyDeactivation(now)
		},
	)
	if err != nil {
		return nil, wrapTenantErr(err)
	}

	if err := s.auditEmitter.emitTenantDeactivated(ctx, models.TenantDeactivated{TenantID: tenant.ID}); err != nil {
		return nil, err
	}

	return tenant, nil
}

// ReactivateTenant transitions a tenant to active status.
// Returns the updated tenant or an error if tenant is not found or already active.
//
// Uses the Execute callback pattern for atomic validate-then-mutate.
// The store's Execute method holds the lock (mutex or FOR UPDATE) during both validation and mutation.
func (s *TenantService) ReactivateTenant(ctx context.Context, tenantID id.TenantID) (*models.Tenant, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}

	now := requestcontext.Now(ctx)
	tenant, err := s.tenants.Execute(ctx, tenantID,
		func(t *models.Tenant) error {
			if err := t.CanReactivate(); err != nil {
				if dErrors.HasCode(err, dErrors.CodeInvariantViolation) {
					return dErrors.New(dErrors.CodeConflict, "tenant is already active")
				}
				return err
			}
			return nil
		},
		func(t *models.Tenant) {
			t.ApplyReactivation(now)
		},
	)
	if err != nil {
		return nil, wrapTenantErr(err)
	}

	if err := s.auditEmitter.emitTenantReactivated(ctx, models.TenantReactivated{TenantID: tenant.ID}); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) incrementTenantCreated() {
	if s.metrics != nil {
		s.metrics.IncrementTenantCreated()
	}
}
