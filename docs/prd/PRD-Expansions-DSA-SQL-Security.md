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

---

## Retrofits for Completed PRDs

The following PRDs are already implemented. These SQL/database enhancements should be applied as technical debt items or during future refactoring:

### PRD-001: Authentication & Session Management (Completed)

**SQL Query Patterns to Retrofit:**

- **Window Functions for Session Analytics:**
  ```sql
  SELECT user_id, session_id, created_at,
         ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at DESC) AS session_rank,
         COUNT(*) OVER (PARTITION BY user_id) AS total_sessions,
         LAG(created_at) OVER (PARTITION BY user_id ORDER BY created_at) AS prev_session_start
  FROM sessions
  WHERE status = 'active';
  ```

- **CTE for Concurrent Session Detection:**
  ```sql
  WITH active_sessions AS (
    SELECT user_id, COUNT(*) AS session_count
    FROM sessions
    WHERE status = 'active' AND expires_at > NOW()
    GROUP BY user_id
    HAVING COUNT(*) > :max_concurrent_sessions
  )
  SELECT s.* FROM sessions s
  JOIN active_sessions a ON s.user_id = a.user_id;
  ```

- **Indexes:** Composite index on `(user_id, status, created_at)` for session lookups

### PRD-001B: Admin User Deletion (Completed)

**SQL Query Patterns to Retrofit:**

- **CTE for Deletion Impact Preview:**
  ```sql
  WITH user_data AS (
    SELECT 'sessions' AS table_name, COUNT(*) AS row_count FROM sessions WHERE user_id = :user_id
    UNION ALL SELECT 'audit_events', COUNT(*) FROM audit_events WHERE user_id = :user_id
    UNION ALL SELECT 'consent_records', COUNT(*) FROM consent_records WHERE user_id = :user_id
  )
  SELECT * FROM user_data WHERE row_count > 0;
  ```

- **Transactional Cascade Delete:**
  ```sql
  BEGIN;
  DELETE FROM sessions WHERE user_id = :user_id;
  DELETE FROM consent_records WHERE user_id = :user_id;
  UPDATE audit_events SET user_id = :pseudonym WHERE user_id = :user_id;
  DELETE FROM users WHERE id = :user_id;
  COMMIT;
  ```

- **Foreign Key Constraints:** Ensure `ON DELETE CASCADE` or `ON DELETE RESTRICT` is configured appropriately

### PRD-002: Consent Management (Completed)

**SQL Query Patterns to Retrofit (extends existing bullets):**

- **CASE for Consent Status:**
  ```sql
  SELECT user_id, purpose,
         CASE
           WHEN revoked_at IS NOT NULL THEN 'revoked'
           WHEN expires_at < NOW() THEN 'expired'
           ELSE 'active'
         END AS status
  FROM consent_records
  WHERE user_id = :user_id;
  ```

- **Semi-Join for Users with Specific Consent:**
  ```sql
  SELECT u.* FROM users u
  WHERE EXISTS (
    SELECT 1 FROM consent_records c
    WHERE c.user_id = u.id
      AND c.purpose = 'registry_check'
      AND c.revoked_at IS NULL
  );
  ```

- **Partial Index:** `CREATE INDEX idx_active_consents ON consent_records (user_id, purpose) WHERE revoked_at IS NULL;`

### PRD-016: Token Lifecycle & Revocation (Mostly Completed)

**SQL Query Patterns to Retrofit (extends existing bullets):**

- **Window Function for Token Refresh Velocity:**
  ```sql
  SELECT token_id, user_id, refreshed_at,
         COUNT(*) OVER (
           PARTITION BY user_id
           ORDER BY refreshed_at
           RANGE BETWEEN INTERVAL '5 minutes' PRECEDING AND CURRENT ROW
         ) AS refreshes_last_5min
  FROM token_refresh_log
  WHERE refreshes_last_5min > :velocity_threshold;
  ```

- **Anti-Join for Orphaned Tokens:**
  ```sql
  SELECT t.id, t.user_id FROM tokens t
  WHERE NOT EXISTS (
    SELECT 1 FROM sessions s WHERE s.id = t.session_id
  );
  ```

### PRD-026A: Tenant & Client Management (Completed)

**SQL Query Patterns to Retrofit:**

- **Aggregate with GROUP BY for Client Stats:**
  ```sql
  SELECT t.id AS tenant_id, t.name,
         COUNT(DISTINCT c.id) AS client_count,
         COUNT(DISTINCT u.id) AS user_count
  FROM tenants t
  LEFT JOIN clients c ON t.id = c.tenant_id
  LEFT JOIN users u ON t.id = u.tenant_id
  GROUP BY t.id, t.name
  ORDER BY user_count DESC;
  ```

- **Row-Level Security Policy:**
  ```sql
  ALTER TABLE clients ENABLE ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation ON clients
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
  ```

- **JSONB for Client Metadata:**
  ```sql
  SELECT id, name, metadata->>'redirect_uris' AS redirect_uris
  FROM clients
  WHERE metadata @> '{"type": "confidential"}';
  ```

---

## How to apply
- Fold these bullets into the relevant PRD acceptance criteria sections.
- Add metrics to validate them: e.g., latency/complexity bounds, `EXPLAIN` screenshots/plan outputs, security audit events emitted.
- Include feature-driven integration scenarios (godog) that exercise the new constraints (e.g., proving RLS blocks cross-tenant reads, proving Merkle proofs verify).
- For completed PRDs, create technical debt tickets to retrofit the SQL patterns during maintenance windows.
