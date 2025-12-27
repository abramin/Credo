# Domain-Driven Design in Credo

This document explains how Credo applies Domain-Driven Design (DDD) and reviews
the current DDD patterns in the tenant module with concrete examples.

## What DDD Is (Brief)

Domain-Driven Design is an approach to software design that puts the business
domain model at the center. The goal is to encode core rules, language, and
invariants directly in code so behavior is predictable and easy to evolve.

Key ideas:

- Use a shared, precise domain language (ubiquitous language).
- Model key concepts as entities and value objects.
- Guard invariants at construction or state transitions.
- Draw clear boundaries (bounded contexts) so models do not leak.
- Keep infrastructure (HTTP, DB) outside the core model.

## How Credo Applies DDD

Credo organizes the codebase by domain modules and keeps the core model isolated
from transport and persistence concerns.

- Bounded contexts are represented by `internal/*` modules (auth, consent,
  decision, evidence, tenant, admin). Each module has its own models, services,
  and store contracts.
- Domain primitives are centralized in `pkg/domain/ids.go`, which defines typed
  IDs like `TenantID` and `ClientID` to prevent accidental mixing of identifiers.
- Domain errors are expressed using stable codes in
  `pkg/domain-errors/domain-errors.go` so domain semantics stay transport-agnostic.
- Ports and adapters keep persistence and transport outside the domain logic.
  Store interfaces in services act as repositories, and HTTP handlers map
  request DTOs into domain commands.

## Tenant Module DDD Review (Aggregates, Invariants, Value Objects)

The tenant module is a good example of current DDD usage. It separates domain
models, commands, and services, and keeps infrastructure at the edges.

### Aggregates and Entities

- `Tenant` is an aggregate root representing the tenant lifecycle and identity.
  See `internal/tenant/models/tenant.go`.
- `Client` is a separate aggregate root referencing its owning tenant by
  `TenantID`. See `internal/tenant/models/client.go`.
- The association is enforced at the service layer by repository queries and
  cross-aggregate checks (for example, tenant must be active to create a client)
  in `internal/tenant/service/client_service.go`.

### Invariants and State Transitions

Invariants are checked in constructors and transition methods to keep the model
valid by default.

- `NewTenant` enforces non-empty and length-bounded names, and initializes
  timestamps and status in `internal/tenant/models/tenant.go`.
- `Tenant.Deactivate` and `Tenant.Reactivate` enforce valid lifecycle
  transitions in `internal/tenant/models/tenant.go`.
- `NewClient` enforces required OAuth fields and consistency in
  `internal/tenant/models/client.go`.
- OAuth-specific rules (URI validation, grant compatibility, allowed scopes)
  live in `internal/tenant/service/commands.go`, keeping HTTP DTOs thin and
  validating domain rules before persistence.
- Cross-aggregate invariants (tenant must be active; public clients cannot use
  `client_credentials`) are enforced in `internal/tenant/service/client_service.go`.

### Value Objects and Domain Primitives

- Statuses and grant types are modeled as typed value objects in
  `internal/tenant/models/value_objects.go` with explicit validation and
  behavior (for example, `GrantType.RequiresConfidentialClient`).
- Typed IDs (`TenantID`, `ClientID`) in `pkg/domain/ids.go` prevent identifier
  confusion and make function signatures self-documenting.

### Domain Services and Repositories

- `TenantService` and `ClientService` are domain services that orchestrate
  entity behavior, enforce invariants, and emit audit events.
  See `internal/tenant/service/tenant_service.go` and
  `internal/tenant/service/client_service.go`.
- Repository interfaces (`TenantStore`, `ClientStore`) define persistence
  contracts in `internal/tenant/service/common.go`, keeping services insulated
  from storage details.

### Boundary Translation (Transport to Domain)

- HTTP requests are normalized and mapped to domain commands in
  `internal/tenant/handler/requests.go`.
- Domain validation happens in command objects in
  `internal/tenant/service/commands.go`, avoiding leakage of HTTP concerns into
  the domain model.

## Additional DDD Examples in Credo

### Consent Module

- `Record` is a consent entity with invariants and lifecycle behavior in
  `internal/consent/models/models.go`.
- `Purpose` and `Status` are value objects defining legal consent purposes and
  lifecycle states in `internal/consent/models/value_objects.go`.

### Decision Module

- `DerivedIdentity` is a domain-level value object that strips PII and carries
  only derived attributes for decisioning in `internal/decision/models.go`.

### Registry Module (Evidence Context)

The Registry module demonstrates subdomain decomposition within a bounded context.
It is organized into two first-class **subdomains** with a **shared kernel**:

```
internal/evidence/registry/domain/
├── shared/              # Shared Kernel
│   └── types.go         # NationalID, Confidence, CheckedAt, ProviderID
├── citizen/             # Citizen Subdomain
│   └── citizen.go       # CitizenVerification aggregate
└── sanctions/           # Sanctions Subdomain
    └── sanctions.go     # SanctionsCheck aggregate
```

**Shared Kernel (`domain/shared/`):**
- `NationalID`: Validated lookup key (6-20 alphanumeric chars)
- `Confidence`: Evidence reliability score (0.0-1.0)
- `CheckedAt`: Verification timestamp with TTL-aware freshness checks
- `ProviderID`: Evidence source identifier

**Citizen Subdomain (`domain/citizen/`):**
- `CitizenVerification` is the aggregate root handling identity verification
- `PersonalDetails` is a value object containing PII (name, DOB, address)
- `VerificationStatus` is a value object with Valid flag and CheckedAt
- Key invariant: Minimized records have empty PersonalDetails (one-way transformation)

**Sanctions Subdomain (`domain/sanctions/`):**
- `SanctionsCheck` is the aggregate root handling compliance screening
- `ListType` is a value object enum (sanctions, pep, watchlist)
- `ListingDetails` is a value object with reason and listed date
- Key invariant: If Listed is true, ListType must be set

**Domain Purity:**
All domain packages follow strict purity rules:
- No I/O (no database, HTTP, filesystem)
- No `context.Context` in function signatures
- No `time.Now()` calls—time is received as parameters from the application layer
- Pure input → output functions, fully testable without mocks

See `internal/evidence/registry/README.md` for complete details.

## Glossary

- Aggregate: A cluster of domain objects treated as a unit for consistency.
  Only the aggregate root is referenced from outside the aggregate.
- Aggregate Root: The main entity responsible for enforcing aggregate rules.
- Entity: A domain object with identity and lifecycle (for example, Tenant).
- Value Object: An immutable domain type identified by its values (for example,
  `TenantStatus`, `GrantType`, `Purpose`).
- Invariant: A business rule that must always hold true.
- Bounded Context: A domain boundary where a model is consistent and owned.
- Domain Service: Stateless domain behavior that does not fit naturally on a
  single entity (for example, `ClientService` orchestration).
- Repository: A persistence abstraction for aggregates (for example, `TenantStore`).
- Domain Primitive: A strongly typed domain value (for example, typed IDs).
- Ubiquitous Language: The shared vocabulary used consistently across code and
  domain discussions.
