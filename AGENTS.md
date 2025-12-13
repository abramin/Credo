## Overview

Credo is a modular identity and evidence platform composed of small, isolated APIs. Each module exposes a handler layer for HTTP, a service layer for business logic, models for data structures, and a store interface for persistence or caching. The goal is clarity, testability, and interchangeable components.

This document defines the conventions used across all Credo modules.

---

## Module Structure Rules

### 1. Handlers

- Only handle HTTP concerns: parsing, validation, converting service outputs to responses.
- No business logic in handlers.
- Always accept and pass through `context.Context`.

### 2. Services

- All business logic lives in the service layer.
- Services depend on stores, clients, and publishers via interfaces.
- Services are responsible for orchestration, validation beyond input shape, and error handling.
- Optimise for unit testing: dependency injection only, no globals.

### 3. Models

- Define pure data structures.
- No business logic.
- Keep domain models separate from transport types when needed.

### 4. Stores

- Interfaces only.
- Allow in-memory or SQL-backed implementations.
- If persistent SQL is used, generate queries using **sqlc** to avoid handwritten SQL and reduce drift.
- Stores return domain models, not DB-specific structs.

---

## Testing Rules

### 1. gomock

- Use gomock for mocking store, client, publisher, or external dependencies.
- Mocks should live under `internal/<module>/mocks`.

### 2. testify

- Use `testify/assert` and `require` for clarity.
- Avoid deep custom comparisons unless necessary.

### 3. BDD-style test structure

Each test follows:

**Given** known state or mocks
**When** the service method is invoked
**Then** assert results, interactions, and errors

Example skeleton:

```
Given(...)
When(...)
Then(...)
```

Helpers may be used for repeated setup inside a module.

### 4. Test suite layout

- Group tests by function: one suite targets one exported method/function.
- Use subtests to cover behaviours and edge cases instead of separate top-level tests.
- Default test contexts should stay minimal; only enable feature flags (e.g., device binding) inside the subtests that exercise them.
- Table tests are preferred for pure validation branches; name cases clearly.

---

## General Principles

- Keep the service layer free from HTTP and DB concerns.
- Use interfaces for any dependency that may need to be mocked or swapped.
- Keep modules independent; no cross-module imports except through interfaces.
- Prefer explicit wiring (constructors) over hidden globals.
- Maintain small, focused files; avoid god objects.
- Refer to docs/architecture.md and the prd folder for details of implementation

## Implementation Patterns

### 1. Service Construction

- Config + options: constructors accept required config plus functional options (e.g., inject logger, JWT service, feature flags) instead of globals.
- Validate required dependencies in constructor; return error if stores or critical config missing.
- Apply sensible defaults for optional fields (TTLs, allowed schemes, etc.).

### 2. Error Handling

- Domain errors: wrap failures with `pkg/domain-errors` codes; prefer `dErrors.Wrap`/`dErrors.New` to keep client-safe messages and telemetry alignment.
- Map store-specific errors (e.g., `sessionStore.ErrNotFound`) to domain error codes (e.g., `dErrors.CodeUnauthorized`) at service boundary.
- Never expose internal implementation details in error messages sent to clients.

### 3. Audit & Observability

- Audit first: emit audit events at key state transitions (user/session/token lifecycle).
- Keep audit publishing inside services, not handlers.
- Include contextual fields: `user_id`, `session_id`, `client_id`, `request_id` when available.
- Use structured logging (slog) with context for correlation; emit security events to both logs and audit streams.

### 4. Context & Middleware

- Middleware data flow: rely on middleware helpers to attach request metadata (client IP, user agent, device ID) into `context.Context`.
- Services read from context rather than accepting parameters for cross-cutting concerns.
- Never store sensitive data in context; use it only for request-scoped metadata and tracing.

### 5. Transactions & State Management

- Transactions: group multi-store writes with `RunInTx` to avoid partial persistence on failure.
- Update session timestamps (`LastSeenAt`, `LastRefreshedAt`) and statuses atomically with token persistence.
- Mark resources as used/consumed (codes, refresh tokens) within the same transaction as token generation.

### 6. Feature Flags & Testing

- Feature flags: default tests/config keep flags off; enable them only in specific scenarios that exercise the feature (e.g., device binding).
- Minimize test boilerplate by keeping default contexts clean; inject feature-specific metadata only where needed.
- Use functional options to enable features, not global state.

### 7. Integration Testing

- Focus on happy-path journeys that verify end-to-end flows required by PRDs.
- Use subtests for error scenarios (400, 401, 404, 500) within the same journey.
- Avoid duplicating unit test coverage; integration tests should validate HTTP-layer wiring, store persistence, and middleware interactions.
- Each integration test suite corresponds to one PRD functional requirement or user journey.

### 8. Mocks & Interfaces

- Keep gomock-generated mocks under `internal/<module>/mocks`.
- Co-locate `//go:generate mockgen` hints with interface definitions for discoverability.
- Regenerate mocks after interface changes and include in same commit.
