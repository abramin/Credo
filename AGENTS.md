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

---

## General Principles

- Keep the service layer free from HTTP and DB concerns.
- Use interfaces for any dependency that may need to be mocked or swapped.
- Keep modules independent; no cross-module imports except through interfaces.
- Prefer explicit wiring (constructors) over hidden globals.
- Maintain small, focused files; avoid god objects.
- Refer to docs/architecture.md and the prd folder for details of implementation
