package providers

import (
	"context"
	"fmt"
	"time"
)

// Protocol defines the supported communication protocols for registry providers
type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolSOAP Protocol = "soap"
	ProtocolGRPC Protocol = "grpc"
)

// ProviderType identifies the kind of evidence a provider can produce
type ProviderType string

const (
	ProviderTypeCitizen   ProviderType = "citizen"
	ProviderTypeSanctions ProviderType = "sanctions"
	ProviderTypeBiometric ProviderType = "biometric"
	ProviderTypeDocument  ProviderType = "document"
	ProviderTypeWallet    ProviderType = "wallet" // Digital ID wallet
)

// FieldCapability advertises which fields a provider exposes
type FieldCapability struct {
	FieldName  string // e.g., "full_name", "date_of_birth", "address"
	Available  bool
	Filterable bool // Whether this field can be used in queries
}

// Capabilities describes what a provider supports
type Capabilities struct {
	Protocol Protocol
	Type     ProviderType
	Fields   []FieldCapability
	Version  string   // Provider API version
	Filters  []string // Supported filter types: "national_id", "passport", "email"
}

// Evidence is the generic result from any provider
type Evidence struct {
	ProviderID   string // Which provider produced this
	ProviderType ProviderType
	Confidence   float64                // 0.0-1.0 confidence score
	Data         map[string]interface{} // Provider-specific structured data
	CheckedAt    time.Time
	Metadata     map[string]string // Provider metadata, trace IDs, etc.
}

// Provider is the universal interface all registry sources must implement
type Provider interface {
	// ID returns a unique identifier for this provider instance
	ID() string

	// Capabilities returns what this provider supports
	Capabilities() Capabilities

	// Lookup performs an evidence check using the provider
	// The input map should contain filter fields (e.g., "national_id", "email")
	Lookup(ctx context.Context, filters map[string]string) (*Evidence, error)

	// Health checks if the provider is available
	Health(ctx context.Context) error
}

// ProviderRegistry maintains all registered providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new empty registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(p Provider) error {
	id := p.ID()
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider %s already registered", id)
	}
	r.providers[id] = p
	return nil
}

// Get retrieves a provider by ID
func (r *ProviderRegistry) Get(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// ListByType returns all providers of a given type
func (r *ProviderRegistry) ListByType(t ProviderType) []Provider {
	var result []Provider
	for _, p := range r.providers {
		if p.Capabilities().Type == t {
			result = append(result, p)
		}
	}
	return result
}

// All returns all registered providers
func (r *ProviderRegistry) All() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}
