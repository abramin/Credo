# PRD-005B: Cerbos-Based Authorization

**Status:** Not Started
**Priority:** P1 (Policy-as-Code)
**Owner:** Engineering Team
**Dependencies:** PRD-005 (Decision Engine), PRD-006 (Audit)
**Last Updated:** 2025-12-06

---

## 1. Overview

### Problem Statement
The current decision engine uses bespoke rule code. It is hard to audit, change, or delegate to non-developers. We need policy-as-code with versioning, testing, and clear separation between enforcement and business logic.

### Goals
- Integrate Cerbos as the policy engine for fine-grained authorization.
- Externalize authorization rules into YAML policies with git-based change control.
- Provide PDP (Cerbos) deployment alongside the gateway with health checks and metrics.
- Ensure existing decision flows can call Cerbos with minimal code changes.
- Add developer tooling for local policy testing.

### Non-Goals
- Multi-tenant Cerbos clusters (single PDP instance is sufficient for MVP).
- UI for policy editing (use files + code review).
- ABAC/PBAC feature parity beyond what Cerbos provides out of the box.

---

## 2. User Stories

**As a backend engineer**
- I want to change authorization rules without redeploying the gateway code.

**As a security/compliance reviewer**
- I want to audit and version control authorization policies.

**As an SRE**
- I want clear health/metrics for the PDP so outages are detected quickly.

**As an integrator**
- I want consistent decision responses (allow/deny + reason) from a stable API.

---

## 3. Functional Requirements

### FR-1: Policy Modeling
- Define Cerbos resource kinds for the gateway (e.g., `auth_session`, `credential`, `consent_record`, `decision_request`).
- Express roles/attributes: user role, regulated mode flag, consent state, evidence signals.
- Store policies under `deploy/policies/` with examples and tests.

### FR-2: PDP Deployment
- Run Cerbos sidecar/container in dev and CI (Docker image). Provide `docker-compose` service and K8s manifest stub.
- Health endpoint `/_cerbos/health` checked on startup; fail fast if unavailable.
- Metrics exposed at `/metrics` and scraped by Prometheus (or logged in dev).

### FR-3: Decision API Integration
- Gateway calls Cerbos `checkResources` API during decision flows (PRD-005) and consent-protected actions.
- Request context includes subject, resource, actions, and attributes (regulated mode, risk score, consent flags).
- Normalize Cerbos responses into existing gateway response format (allow/deny, reasons, obligations).

### FR-4: Policy Tests & CI
- Add Cerbos policy tests (YAML) runnable via `cerbos compile --tests` or equivalent.
- CI step executes policy tests and blocks on failure.
- Provide sample fixtures for common scenarios (happy path, missing consent, sanctions hit, high risk score).

### FR-5: Observability & Audit
- Log every Cerbos decision with request ID, subject, resource, action, outcome, latency.
- Emit metrics: decision count, latency histogram, error count, PDP availability.
- Audit events include policy version/sha.

### FR-6: Backward Compatibility & Fallbacks
- If Cerbos is down, fail closed (deny) with explicit error code; configurable fail-open for local dev only.
- Maintain the existing rule engine behind a feature flag as fallback during rollout.

---

## 4. Acceptance Criteria
- Cerbos runs locally via docker-compose and in CI; health check passes before integration tests.
- Policies for at least: creating sessions, issuing credentials, registry lookups, consent-required actions, and decision engine evaluations.
- Gateway decision path uses Cerbos response to allow/deny; responses include reason/obligations for UI/logging.
- Policy tests exist and run in CI; failing tests block merges.
- Metrics and logs show Cerbos decision latency and counts; audit events include policy version.

---

## 5. Risks & Open Questions
- Policy drift between environments; need clear deployment/versioning story.
- Performance overhead of external PDP calls under load.
- Strategy for policy migration from legacy rules; dual-run period may be needed.
