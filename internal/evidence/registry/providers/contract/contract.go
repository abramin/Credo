package contract

import (
	"context"
	"encoding/json"
	"testing"

	"credo/internal/evidence/registry/providers"
)

// ContractTest defines a test case for provider contract validation
type ContractTest struct {
	Name         string
	Provider     providers.Provider
	Input        map[string]string
	ExpectedType providers.ProviderType
	ValidateFunc func(evidence *providers.Evidence) error
}

// ContractSuite is a collection of contract tests for a provider
type ContractSuite struct {
	ProviderID      string
	ProviderVersion string
	Tests           []ContractTest
}

// Run executes all contract tests in the suite
func (s *ContractSuite) Run(t *testing.T) {
	for _, test := range s.Tests {
		t.Run(test.Name, func(t *testing.T) {
			ctx := context.Background()

			// Execute lookup
			evidence, err := test.Provider.Lookup(ctx, test.Input)
			if err != nil {
				t.Fatalf("provider lookup failed: %v", err)
			}

			// Validate provider ID matches
			if evidence.ProviderID != s.ProviderID {
				t.Errorf("expected provider ID %s, got %s", s.ProviderID, evidence.ProviderID)
			}

			// Validate provider type
			if evidence.ProviderType != test.ExpectedType {
				t.Errorf("expected type %s, got %s", test.ExpectedType, evidence.ProviderType)
			}

			// Validate confidence is in valid range
			if evidence.Confidence < 0 || evidence.Confidence > 1.0 {
				t.Errorf("confidence %f out of range [0, 1]", evidence.Confidence)
			}

			// Validate CheckedAt is set
			if evidence.CheckedAt.IsZero() {
				t.Error("CheckedAt not set")
			}

			// Run custom validation
			if test.ValidateFunc != nil {
				if err := test.ValidateFunc(evidence); err != nil {
					t.Errorf("custom validation failed: %v", err)
				}
			}
		})
	}
}

// SnapshotTest compares provider output against saved snapshot
type SnapshotTest struct {
	Name     string
	Provider providers.Provider
	Input    map[string]string
	Snapshot string // Path to saved snapshot JSON
}

// Run executes a snapshot test
func (st *SnapshotTest) Run(t *testing.T) {
	ctx := context.Background()

	// Execute lookup
	evidence, err := st.Provider.Lookup(ctx, st.Input)
	if err != nil {
		t.Fatalf("provider lookup failed: %v", err)
	}

	// Serialize to JSON for comparison
	actualJSON, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal evidence: %v", err)
	}

	// TODO: Load snapshot and compare
	// For now, just log the output
	t.Logf("Evidence snapshot:\n%s", string(actualJSON))
}

// CapabilityTest validates that provider capabilities are correctly declared
type CapabilityTest struct {
	Provider providers.Provider
}

// Run executes a capability test
func (ct *CapabilityTest) Run(t *testing.T) {
	caps := ct.Provider.Capabilities()

	// Validate protocol is set
	if caps.Protocol == "" {
		t.Error("protocol not set")
	}

	// Validate type is set
	if caps.Type == "" {
		t.Error("type not set")
	}

	// Validate version is set
	if caps.Version == "" {
		t.Error("version not set")
	}

	// Validate fields are declared
	if len(caps.Fields) == 0 {
		t.Error("no field capabilities declared")
	}

	// Validate at least one filter is supported
	if len(caps.Filters) == 0 {
		t.Error("no filters declared")
	}

	t.Logf("Provider %s capabilities:", ct.Provider.ID())
	t.Logf("  Protocol: %s", caps.Protocol)
	t.Logf("  Type: %s", caps.Type)
	t.Logf("  Version: %s", caps.Version)
	t.Logf("  Fields: %d", len(caps.Fields))
	t.Logf("  Filters: %v", caps.Filters)
}

// ErrorContractTest validates that provider errors follow the taxonomy
type ErrorContractTest struct {
	Name          string
	Provider      providers.Provider
	Input         map[string]string
	ExpectedError providers.ErrorCategory
	ExpectedRetry bool
}

// Run executes an error contract test
func (ect *ErrorContractTest) Run(t *testing.T) {
	ctx := context.Background()

	_, err := ect.Provider.Lookup(ctx, ect.Input)
	if err == nil {
		t.Fatal("expected error but got none")
	}

	// Validate error is a ProviderError
	category := providers.GetCategory(err)
	if category != ect.ExpectedError {
		t.Errorf("expected error category %s, got %s", ect.ExpectedError, category)
	}

	// Validate retry flag
	isRetryable := providers.IsRetryable(err)
	if isRetryable != ect.ExpectedRetry {
		t.Errorf("expected retryable=%v, got %v", ect.ExpectedRetry, isRetryable)
	}
}
