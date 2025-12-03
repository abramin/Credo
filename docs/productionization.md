# Productionisation Considerations

This section outlines the changes required to evolve the Identity Verification Gateway from a functional prototype into a production-ready service suitable for regulated domains such as fintech, identity verification, or healthcare. The goal is not to implement everything immediately, but to demonstrate awareness of real-world constraints and the architectural pathways to support them.

## 1. Replace In-Memory Stores With Durable Backends

**Current State:**
All stores (user, session, consent, VC, registry cache, audit) are in-memory maps.

**Required Changes:**

* Migrate to Postgres (transactional, ACID, audit-friendly).
* Add schema migrations via Goose or Atlas.
* Introduce typed repositories with SQLC or Ent.
* Ensure strict foreign-key relationships (user → consent, VC, audit).
* Add optimistic locking where needed (e.g., VC revocation).

**Rationale:**
Durability, auditability, stronger guarantees, and ability to handle multi-instance scaling.

---

## 2. Introduce a Real Queue for the Audit Pipeline

**Current State:**
Audit events are stored synchronously in-memory.

**Required Changes:**

* Publish audit events to a queue (NATS, Kafka, or AWS SQS).
* Add a background audit worker that drains messages and writes to an `AuditStore` (Postgres).
* Introduce DLQ (dead letter queue) for failed or malformed audit events.
* Include request IDs and correlation IDs in event metadata.

**Rationale:**
Regulated systems must never lose audit trails. The queue decouples request latency from audit persistence and improves reliability.

---

## 3. Real Cryptography for Verifiable Credentials

**Current State:**
VC issuance uses mock signatures.

**Required Changes:**

* Use JOSE libraries (JWS/JWT) or BBS+ signatures (for selective disclosure).
* Add key rotation, KMS-integrated key storage.
* Implement issuer DID (did:web or did:key).
* Add a revocation registry (list or status endpoint).

**Rationale:**
Authenticity and tamper-proofing of credentials is mandatory in real digital ID flows.

---

## 4. Security Hardening 

**Must-have improvements:**

* OIDC provider replaced with a real provider (Hydra, Keycloak, Auth0).
* End-to-end HTTPS/TLS enforcement (including internal registries).
* JWT validation, refresh tokens, token rotation.
* Rate limiting and WAF integration (e.g., Cloudflare or Envoy).
* Mutual TLS or signed requests between services (registry calls).
* Threat modelling: replay protections, CSRF for UI, anti-session fixation, timing-attack safe comparisons.

**Rationale:**
Authentication systems are high-value targets. Hardening and defence-in-depth are expected.

---

## 5. Consent Lifecycle Strengthening

**Current State:**
Basic consent model with purpose and expiry.

**Required Changes:**

* Versioned consent policies (e.g., regulatory updates over time).
* Explicit consent scopes mapped to operational flows.
* User consent receipt generation (PDF or signed JSON).
* Persistent audit logs for grant, revoke, expiry.
* Integration with privacy team for DPIA considerations.

**Rationale:**
Regulated environments require traceable, legally demonstrable consent.

---

## 6. Registry Integration Protocols

**Current State:**
Mocked citizen and sanctions registries.

**Required Changes:**

* External registry calls via signed requests or OAuth2 client credentials.
* SLA-based retry logic and circuit breakers (Hystrix-style).
* Domain-specific timeouts and isolation (per registry).
* Caching with controlled TTL and integrity checks.
* Real-time monitoring of third-party failure rates.

**Rationale:**
Registry dependencies are often slow or unreliable. Isolation protects user-facing performance.

---

## 7. Observability & Operations

**Current State:**
Minimal logging.

**Required Changes:**

* Structured logs with correlation IDs.
* Metrics (Prometheus): latency, error rates, registry call durations, queue lag.
* Tracing (OpenTelemetry): distributed spans across login → consent → registry → decision.
* Dashboards for live monitoring (Grafana).
* Alerts for audit worker errors, queue lag, registry failures.

**Rationale:**
Regulated systems need strong operational awareness and incident response capabilities.

---

## 8. Policy & Data Retention Enforcement

**Current State:**
TTL on registry caches only.

**Required Changes:**

* Policy engine (OPA/Cerbos) or custom rule layer to enforce retention, data flows, and purpose restrictions.
* Configurable retention per data type.
* Automated deletion jobs (cron or queue-driven).
* Immutable audit log storage (WORM storage such as AWS Glacier or S3 Object Lock).
* Privacy-by-default: derived attributes preferred over raw PII.

**Rationale:**
Retention policies are legally binding. Evidence of enforcement must be auditable.

---

## 9. Deployment & Scaling

**Changes needed:**

* Containerisation (Docker) with multi-stage builds.
* Kubernetes deployment (simple Helm chart).
* Rolling updates and health checks.
* Horizontal scaling for the API; separate scaling for the audit worker.
* Optional: use of a service mesh (Linkerd, Istio) for mTLS and traffic shaping.

**Rationale:**
Identity services must scale independently across critical paths and background tasks.

---

## 10. Disaster Recovery & Fault Tolerance

**Additions:**

* Database replicas + PITR backups.
* Geo-reduntant deployments if multi-region is needed.
* Queue replication or use of managed durable queues.
* Runbook for registry outage handling.
* Graceful degradation modes (e.g., cached-only verification, VC issuance disabled).

**Rationale:**
Regulated environments require high availability and predictable behaviour during government or registry outages.

---

## 11. Compliance Alignment

To signal deep alignment with identity systems:

* Map flows to **eIDAS2**, **NIST 800-63-3**, **ETSI TS 119** standards.
* Produce a privacy data map showing all PII categories and retention rules.
* Add threat models for verification flows.
* Add conformance tests for VC formats.

**Rationale:**
Shows that you can speak to auditors, compliance, and risk teams.

---

## 12. Web UI Productionisation

**Changes:**

* Hard secrets removed; backend URLs configurable.
* CSRF protection.
* Strict Content Security Policy.
* Session storage minimisation.
* Accessibility standards (WCAG 2.1 AA).
* Audit logs for user actions in UI (consent screens especially).

**Rationale:**
Trust flows are user-facing. Accessibility and security matter.