# DSA, SQL, and Security-First Expansion Opportunities

This note highlights places to strengthen existing PRDs with explicit requirements that surface data-structures/algorithms practice, SQL depth, and secure-by-design patterns (per “Secure by Design” by Dan Bergh). Each item calls out the PRD and specific additions to bake into the requirements/acceptance criteria.

## Targeted PRD Additions

- **PRD-017 Rate Limiting & Abuse Prevention (DSA + SQL + Security)**
  - Add acceptance criteria for a sliding-window deque implementation plus a bucketed time-wheel alternative, both with O(1) amortized operations and documented complexity proofs.
  - Require a Postgres-backed option using `INSERT ... ON CONFLICT` and hash partitioning by key; include `EXPLAIN`-verified indexes on `(key, window_end)`.
  - Security: mandate atomic multi-key resets (transactions) and explicit abuse thresholds that trigger lockouts and audit events.

- **PRD-005 Decision Engine (DSA + SQL + Security)**
  - Require rule graph evaluation via topological sort (DAG) with cycle detection and memoization; document worst-case complexity and cache eviction strategy (LRU).
  - Persist rules in normalized tables with constraints on versioning, immutability of published rules, and `CHECK` constraints for bounds.
  - Security: signed policy bundles with audit trails on publish, plus deterministic execution to prevent side-channel leaks from timing variability.

- **PRD-006 / PRD-006B Audit & Cryptographic Audit (DSA + SQL + Security)**
  - Specify Merkle tree or append-only log with periodic root anchoring; include proof generation and verification APIs.
  - SQL: partitioned audit tables (by day/week) with covering indexes for `actor, action, ts`; require `EXPLAIN` plans and retention + WORM posture.
  - Security: integrity checks on ingestion (HMAC/signature), tamper-evident digests stored separately, and least-privilege readers.

- **PRD-011 Internal TCP Event Ingester (DSA + SQL + Security)**
  - Add bounded MPSC ring buffer with backpressure and drop/newest vs drop/oldest strategies; require load-shed thresholds and metrics for queue depth.
  - SQL: batch inserts with COPY, idempotency keys, and retries with exponential backoff; acceptance criterion to show reduced write amplification.
  - Security: authenticated producer connections, poison-message quarantine, and replay protection via sequence numbers.

- **PRD-002 Consent Management + PRD-007 Data Rights (SQL + Security)**
  - SQL: migration requirements for row-level security (tenant/user scoping), partial indexes on `(user_id, purpose, status)`, and CQRS read-model projections with `EXPLAIN` outputs.
  - Security: immutable consent event log with hash chaining, and redaction paths that prove PII fields are nulled/removed within a bounded SLA.

- **PRD-016 Token Lifecycle & Revocation (DSA + Security)**
  - Require revocation list backed by a radix/prefix tree for JTIs to allow prefix revocation of device families; specify O(log n) lookup target.
  - Security: mandatory constant-time token comparisons and key rotation drills; acceptance tests for replay detection under concurrent refresh storms.

- **PRD-023 Fraud Detection & Security Intelligence (DSA + SQL)**
  - Add streaming feature-store backed by Count-Min Sketches/Bloom filters for velocity checks; specify false-positive bounds.
  - SQL: materialized views for aggregated risk signals with scheduled refresh and indexed hot columns; require explain plans under load.

- **PRD-026 Admin Dashboard & Operations UI (SQL + Security)**
  - SQL: scoped admin queries with RLS and parameterized search; precomputed summaries via views instead of ad-hoc wide joins.
  - Security: explicit admin token acquisition flow, short-lived session tokens, CSRF defenses, and audit hooks for every privileged action.

- **PRD-025 Developer Sandbox & Testing (DSA + SQL + Security)**
  - Include kata-style exercises: implement LRU cache and rate limiter variants behind feature flags to compare behaviors; provide fixtures.
  - SQL: sandbox migrations with intentional query anti-patterns and required fixes (indexing, N+1 elimination) validated via `EXPLAIN`.
  - Security: safe defaults (deny-by-default network egress, sealed secrets), and mutation testing to ensure input validation paths are covered.

- **PRD-019 API Versioning & Lifecycle (Security)**
  - Require schema-diff gates that block unsafe changes, with signed migration manifests and rollback drills; publish deprecation headers with security impact notes.

## How to apply
- Fold these bullets into the relevant PRD acceptance criteria sections.
- Add metrics to validate them: e.g., latency/complexity bounds, `EXPLAIN` screenshots/plan outputs, security audit events emitted.
- Include feature-driven integration scenarios (godog) that exercise the new constraints (e.g., proving RLS blocks cross-tenant reads, proving Merkle proofs verify).
