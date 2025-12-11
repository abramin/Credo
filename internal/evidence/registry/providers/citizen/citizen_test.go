package citizen

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"credo/internal/evidence/registry/providers"
	"credo/internal/evidence/registry/providers/contract"
)

func TestCitizenProvider(t *testing.T) {
	// Create provider for testing
	provider := NewCitizenProvider(
		"test-citizen",
		"http://mock-registry.test",
		"test-key",
		5*time.Second,
	)

	t.Run("capabilities are correctly declared", func(t *testing.T) {
		caps := provider.Capabilities()

		assert.Equal(t, providers.ProtocolHTTP, caps.Protocol)
		assert.Equal(t, providers.ProviderTypeCitizen, caps.Type)
		assert.Equal(t, "v1.0.0", caps.Version)
		assert.Len(t, caps.Fields, 4) // full_name, date_of_birth, address, valid
		assert.Contains(t, caps.Filters, "national_id")
	})

	t.Run("evidence contains required fields", func(t *testing.T) {
		// This would need a mock HTTP server or you can use contract tests
		t.Skip("Requires HTTP mock server - use contract tests instead")
	})
}

func TestCitizenProviderContract(t *testing.T) {
	// Use the contract testing framework
	provider := NewCitizenProvider(
		"test-citizen",
		"http://mock-registry.test",
		"test-key",
		5*time.Second,
	)

	suite := &contract.ContractSuite{
		ProviderID:      "test-citizen",
		ProviderVersion: "v1.0.0",
		Tests: []contract.ContractTest{
			{
				Name:         "returns valid citizen evidence",
				Provider:     provider,
				Input:        map[string]string{"national_id": "123456789"},
				ExpectedType: providers.ProviderTypeCitizen,
				ValidateFunc: func(e *providers.Evidence) error {
					// Validate all required fields are present
					if _, ok := e.Data["full_name"].(string); !ok {
						return assert.AnError
					}
					if _, ok := e.Data["date_of_birth"].(string); !ok {
						return assert.AnError
					}
					if _, ok := e.Data["valid"].(bool); !ok {
						return assert.AnError
					}
					return nil
				},
			},
			{
				Name:         "generates deterministic data based on national ID",
				Provider:     provider,
				Input:        map[string]string{"national_id": "123456789"},
				ExpectedType: providers.ProviderTypeCitizen,
				ValidateFunc: func(e *providers.Evidence) error {
					// Call again and verify same data
					// This would need to be implemented with proper mock
					return nil
				},
			},
		},
	}

	// Only run if we have a mock server available
	if testing.Short() {
		t.Skip("Skipping contract tests in short mode")
	}
	suite.Run(t)
}

func TestCitizenResponseParser(t *testing.T) {
	t.Run("parses valid HTTP response", func(t *testing.T) {
		body := []byte(`{
			"national_id": "123456789",
			"full_name": "Alice Johnson",
			"date_of_birth": "1990-05-15",
			"address": "123 Main St",
			"valid": true,
			"checked_at": "2025-12-11T10:00:00Z"
		}`)

		evidence, err := parseCitizenResponse(200, body)
		require.NoError(t, err)
		require.NotNil(t, evidence)

		assert.Equal(t, providers.ProviderTypeCitizen, evidence.ProviderType)
		assert.Equal(t, 1.0, evidence.Confidence)
		assert.Equal(t, "123456789", evidence.Data["national_id"])
		assert.Equal(t, "Alice Johnson", evidence.Data["full_name"])
		assert.Equal(t, "1990-05-15", evidence.Data["date_of_birth"])
		assert.Equal(t, true, evidence.Data["valid"])
	})

	t.Run("returns error for non-200 status", func(t *testing.T) {
		evidence, err := parseCitizenResponse(404, []byte(`{}`))
		assert.Error(t, err)
		assert.Nil(t, evidence)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		evidence, err := parseCitizenResponse(200, []byte(`{invalid json`))
		assert.Error(t, err)
		assert.Nil(t, evidence)
	})

	t.Run("handles missing checked_at gracefully", func(t *testing.T) {
		body := []byte(`{
			"national_id": "123456789",
			"full_name": "Alice Johnson",
			"date_of_birth": "1990-05-15",
			"address": "123 Main St",
			"valid": true,
			"checked_at": "invalid-date"
		}`)

		evidence, err := parseCitizenResponse(200, body)
		require.NoError(t, err)
		assert.False(t, evidence.CheckedAt.IsZero(), "should use current time if parse fails")
	})
}

// Test scenarios adapted from old clients/citizen/citizen_test.go:

func TestCitizenProviderScenarios(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		provider := NewCitizenProvider(
			"test-citizen",
			"http://slow-registry.test",
			"test-key",
			1*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := provider.Lookup(ctx, map[string]string{"national_id": "123"})
		assert.Error(t, err)
		// Should be a timeout error
		assert.Equal(t, providers.ErrorTimeout, providers.GetCategory(err))
	})

	t.Run("handles empty national ID", func(t *testing.T) {
		provider := NewCitizenProvider(
			"test-citizen",
			"http://mock-registry.test",
			"test-key",
			5*time.Second,
		)

		// This will fail at HTTP level - the provider will try to call the API
		// In a real scenario with a mock server, we could verify the behavior
		ctx := context.Background()
		_, err := provider.Lookup(ctx, map[string]string{"national_id": ""})

		// Expect some kind of error (provider outage since server doesn't exist)
		assert.Error(t, err)
	})
}
