package registry

import (
	"time"

	"credo/internal/evidence/registry/orchestrator"
	"credo/internal/evidence/registry/orchestrator/correlation"
	"credo/internal/evidence/registry/providers"
	"credo/internal/evidence/registry/providers/citizen"
	"credo/internal/evidence/registry/providers/sanctions"
)

// WiringExample demonstrates how to wire up the new architecture
func WiringExample() *orchestrator.Orchestrator {
	// 1. Create provider registry
	registry := providers.NewProviderRegistry()

	// 2. Register providers
	// Mock citizen registry
	citizenProv := citizen.NewCitizenProvider(
		"mock-citizen-gov",
		"http://mock-citizen-registry.gov/api",
		"secret-api-key",
		5*time.Second,
	)
	_ = registry.Register(citizenProv)

	// Alternative citizen source (e.g., civil registry)
	civilProv := citizen.NewCitizenProvider(
		"civil-registry",
		"http://civil-registry.gov/api",
		"another-key",
		5*time.Second,
	)
	_ = registry.Register(civilProv)

	// Sanctions provider
	sanctionsProv := sanctions.NewSanctionsProvider(
		"sanctions-db",
		"http://sanctions-db.int/api",
		"sanctions-key",
		3*time.Second,
	)
	_ = registry.Register(sanctionsProv)

	// 3. Define provider chains with fallback logic
	chains := map[providers.ProviderType]orchestrator.ProviderChain{
		providers.ProviderTypeCitizen: {
			Primary:   "mock-citizen-gov",
			Secondary: []string{"civil-registry"}, // Fallback if primary fails
			Timeout:   5 * time.Second,
		},
		providers.ProviderTypeSanctions: {
			Primary:   "sanctions-db",
			Secondary: []string{}, // No fallback for sanctions
			Timeout:   3 * time.Second,
		},
	}

	// 4. Define correlation rules for multi-source scenarios
	rules := []orchestrator.CorrelationRule{
		&correlation.CitizenNameRule{},
		&correlation.WeightedAverageRule{
			Weights: map[providers.ProviderType]float64{
				providers.ProviderTypeCitizen:   0.8,
				providers.ProviderTypeSanctions: 1.0,
			},
		},
	}

	// 5. Create orchestrator
	orch := orchestrator.NewOrchestrator(orchestrator.OrchestratorConfig{
		Registry:        registry,
		DefaultStrategy: orchestrator.StrategyFallback,
		DefaultTimeout:  10 * time.Second,
		Chains:          chains,
		Rules:           rules,
	})

	return orch
}

// ServiceExample shows how the service layer uses the orchestrator
/*
func (s *Service) Check(ctx context.Context, nationalID string) (*models.RegistryResult, error) {
    // Use orchestrator to gather all evidence
    result, err := s.orchestrator.Lookup(ctx, orchestrator.LookupRequest{
        Types: []providers.ProviderType{
            providers.ProviderTypeCitizen,
            providers.ProviderTypeSanctions,
        },
        Filters: map[string]string{
            "national_id": nationalID,
        },
        Strategy: orchestrator.StrategyFallback,
    })

    if err != nil {
        return nil, err
    }

    // Convert Evidence to models.CitizenRecord and models.SanctionsRecord
    var citizen *models.CitizenRecord
    var sanction *models.SanctionsRecord

    for _, evidence := range result.Evidence {
        switch evidence.ProviderType {
        case providers.ProviderTypeCitizen:
            citizen = convertEvidenceToCitizen(evidence)
        case providers.ProviderTypeSanctions:
            sanction = convertEvidenceToSanctions(evidence)
        }
    }

    // Apply caching, minimization, audit logging as before

    return &models.RegistryResult{
        Citizen:  citizen,
        Sanction: sanction,
    }, nil
}
*/
