package ports

import "context"

// ConsentPort defines the interface for consent checks
// This is a hexagonal architecture port - the domain layer depends on this interface,
// and adapters (gRPC client, HTTP client, mock) implement it.
//
// This keeps the registry service independent of:
// - gRPC implementation details
// - Protobuf types
// - HTTP/JSON marshaling
// - External service locations
type ConsentPort interface {
	// HasConsent checks if a user has active consent for a purpose
	// Returns true if consent is active, false otherwise
	HasConsent(ctx context.Context, userID string, purpose string) (bool, error)

	// RequireConsent enforces consent requirement
	// Returns nil if consent is active, error otherwise
	// Error types should match pkg/errors conventions
	RequireConsent(ctx context.Context, userID string, purpose string) error
}
