// Package domain contains the pure domain model for the Registry bounded context.
//
// # Registry Bounded Context
//
// The Registry context is responsible for gathering identity evidence from external
// sources such as national population registries and sanctions databases. It provides
// a unified abstraction over heterogeneous evidence providers.
//
// # Subdomain Structure
//
// The Registry bounded context is organized into two distinct subdomains:
//
//	registry/domain/
//	├── shared/     # Shared Kernel - common types used across subdomains
//	├── citizen/    # Citizen Subdomain - identity verification from population registries
//	└── sanctions/  # Sanctions Subdomain - compliance screening against watchlists
//
// # Shared Kernel (shared/)
//
// Contains domain primitives shared between Citizen and Sanctions subdomains:
//   - NationalID: Validated lookup key for registry queries
//   - Confidence: Evidence reliability score (0.0-1.0)
//   - CheckedAt: Verification timestamp with TTL checking
//   - ProviderID: Evidence source identifier
//
// # Citizen Subdomain (citizen/)
//
// Handles identity verification through national population registries.
//
// Aggregate Root: CitizenVerification
//   - Contains PII (name, DOB, address) that requires GDPR compliance
//   - Supports minimization for regulated environments
//   - Tracks verification status and provenance
//
// Key Invariants:
//   - NationalID is always present and valid
//   - Minimized records have empty PersonalDetails
//   - Cannot "un-minimize" a minimized record (immutable transformation)
//
// # Sanctions Subdomain (sanctions/)
//
// Handles compliance screening against sanctions lists and PEP databases.
//
// Aggregate Root: SanctionsCheck
//   - Determines list membership (sanctions, PEP, watchlist)
//   - Provides listing details (reason, date, source)
//   - No PII minimization needed (only contains boolean flags and metadata)
//
// Key Invariants:
//   - Source is always present (authoritative source of check)
//   - If Listed is true, ListType must be set
//   - If Listed is false, ListingDetails is empty
//
// # Domain Purity
//
// All packages in this module follow strict domain purity rules:
//
//	✓ No I/O (no database, HTTP, filesystem access)
//	✓ No context.Context in function signatures
//	✓ No time.Now() or rand.* calls - time is received as parameters
//	✓ Pure input → output functions, fully testable without mocks
//
// The application layer (service/) is responsible for:
//   - Injecting current time for TTL checks
//   - Coordinating with infrastructure (providers, cache, audit)
//   - Translating between domain types and infrastructure types
//
// # Relationship to Infrastructure
//
// Domain types are distinct from infrastructure types:
//
//	Domain (pure)              Infrastructure (effectful)
//	─────────────────────      ─────────────────────────────
//	citizen.CitizenVerification   →   models.CitizenRecord
//	sanctions.SanctionsCheck      →   models.SanctionsRecord
//	shared.NationalID             →   providers.Evidence.Data
//
// The service layer contains converters that translate between these layers,
// keeping the domain pure and the infrastructure concerns isolated.
package domain
