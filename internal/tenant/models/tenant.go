package models

import (
	"time"

	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
)

// Tenant is the aggregate root for a tenant organization.
//
// Invariants:
//   - Name is non-empty and at most 128 characters
//   - Status is either active or inactive
//   - Status transitions: active ↔ inactive only (no other states)
//   - CreatedAt is immutable after construction
//
// # Cascade Invariant
//
// When a tenant is deactivated, all OAuth flows for its clients MUST fail,
// even if the client itself has Status=active. This is enforced at the
// service layer (ResolveClient) rather than by cascading status changes.
//
// Security Implications:
//   - Tenant deactivation is an immediate security boundary enforcement
//   - Clients do NOT need explicit deactivation when tenant is inactive
//   - ResolveClient MUST check tenant.IsActive() before returning client
//   - This prevents suspended organizations from issuing new tokens
//   - Existing tokens remain valid until expiry (revoke separately if needed)
//
// This design choice:
//   - Avoids expensive cascade updates to all clients on tenant status change
//   - Provides single point of enforcement (ResolveClient)
//   - Allows easy reactivation without touching client records
//   - Maintains audit trail clarity (tenant status is the source of truth)
type Tenant struct {
	ID        id.TenantID  `json:"id"`
	Name      string       `json:"name"`
	Status    TenantStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// CanDeactivate checks if the tenant can transition to inactive status.
// Returns an error if the transition is not allowed.
// Use with ApplyDeactivation in Execute callbacks for proper separation of concerns.
func (t *Tenant) CanDeactivate() error {
	if !t.Status.CanTransitionTo(TenantStatusInactive) {
		return dErrors.New(dErrors.CodeInvariantViolation, "tenant is already inactive")
	}
	return nil
}

// ApplyDeactivation transitions the tenant to inactive status.
// Updates the timestamp to track when the transition occurred.
// Call CanDeactivate first to validate the transition.
func (t *Tenant) ApplyDeactivation(now time.Time) {
	t.Status = TenantStatusInactive
	t.UpdatedAt = now
}

// Deactivate validates and applies deactivation in one call.
// Prefer CanDeactivate + ApplyDeactivation for Execute callback pattern.
func (t *Tenant) Deactivate(now time.Time) error {
	if err := t.CanDeactivate(); err != nil {
		return err
	}
	t.ApplyDeactivation(now)
	return nil
}

// CanReactivate checks if the tenant can transition to active status.
// Returns an error if the transition is not allowed.
// Use with ApplyReactivation in Execute callbacks for proper separation of concerns.
func (t *Tenant) CanReactivate() error {
	if !t.Status.CanTransitionTo(TenantStatusActive) {
		return dErrors.New(dErrors.CodeInvariantViolation, "tenant is already active")
	}
	return nil
}

// ApplyReactivation transitions the tenant to active status.
// Updates the timestamp to track when the transition occurred.
// Call CanReactivate first to validate the transition.
func (t *Tenant) ApplyReactivation(now time.Time) {
	t.Status = TenantStatusActive
	t.UpdatedAt = now
}

// Reactivate validates and applies reactivation in one call.
// Prefer CanReactivate + ApplyReactivation for Execute callback pattern.
func (t *Tenant) Reactivate(now time.Time) error {
	if err := t.CanReactivate(); err != nil {
		return err
	}
	t.ApplyReactivation(now)
	return nil
}

func NewTenant(tenantID id.TenantID, name string, now time.Time) (*Tenant, error) {
	if name == "" {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "tenant name cannot be empty")
	}
	if len(name) > 128 {
		return nil, dErrors.New(dErrors.CodeInvariantViolation, "tenant name must be 128 characters or less")
	}
	return &Tenant{
		ID:        tenantID,
		Name:      name,
		Status:    TenantStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
