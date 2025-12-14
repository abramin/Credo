package tenant

import (
	"log/slog"

	"credo/internal/tenant/handler"
	"credo/internal/tenant/service"
)

// Service exposes tenant and client orchestration.
type Service = service.Service

// Handler wires HTTP endpoints to the tenant service.
type Handler = handler.Handler

// NewService constructs the tenant service with required dependencies.
func NewService(tenants service.TenantStore, clients service.ClientStore, users service.UserCounter) *Service {
	return service.New(tenants, clients, users)
}

// NewHandler constructs an HTTP handler for admin-facing tenant routes.
func NewHandler(s *Service, logger *slog.Logger) *Handler {
	return handler.New(s, logger)
}
