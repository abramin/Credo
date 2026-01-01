# PRD-020: Operational Readiness & SRE

**Status:** Not Started
**Priority:** P0 (Critical)
**Owner:** Engineering Team
**Dependencies:** PRD-006 (Audit), all core PRDs
**Last Updated:** 2026-01-01

---

## 1. Overview

### Problem Statement

The system cannot be deployed to production without operational tooling:

- No health check endpoints (Kubernetes liveness/readiness probes fail)
- No backup/restore procedures
- No disaster recovery plan
- No incident response runbooks
- No capacity planning guidelines
- No performance SLAs

### Goals

- Health check endpoints (`/health`, `/ready`, `/live`)
- Liveness vs readiness probes (Kubernetes-compatible)
- Backup & restore procedures
- Disaster recovery plan (RTO/RPO targets)
- Incident response runbooks
- On-call playbooks
- Capacity planning guidelines
- Performance SLAs
- Metrics and alerting baseline for auth, revocation, audit, rate limits, and consent enforcement

### Non-Goals

- Full SRE team hiring plan
- Chaos engineering / fault injection
- Cost optimization strategies

---

## 1B. Storage Infrastructure Transition

### In-Memory First Philosophy

The codebase is intentionally designed with **in-memory stores first, production storage later**. This approach:

- Enables rapid iteration during Phase 0-1 (no external dependencies)
- Keeps tests fast and deterministic
- Uses interfaces throughout, making swapping implementations trivial
- Defers infrastructure complexity until proven necessary

### When to Introduce Production Storage

| Trigger | Required Tool | Rationale |
|---------|--------------|-----------|
| **Multi-instance deployment** | Redis | Rate limiting (PRD-017) must be distributed; in-memory state isn't shared across instances |
| **Data durability requirements** | PostgreSQL | Audit logs (PRD-006), user data, consent records must survive restarts |
| **Compliance/regulatory** | PostgreSQL | GDPR data export requires persistent storage; auditors need durable records |
| **Session sharing** | Redis | Session validation across instances requires shared session store |
| **Backup/DR requirements** | PostgreSQL | Can't backup in-memory stores; `pg_dump` enables point-in-time recovery |
| **Production health checks** | All three | `/health/ready` checks database/Redis/Kafka connectivity (this PRD) |
| **Audit event decoupling** | Kafka | Decouple audit publishing from request latency; guaranteed delivery via outbox pattern (PRD-006 TR-5) |

### Transition Timeline

```
Phase 0-1 (MVP Development)     → In-memory stores only
Phase 2 (Operational Baseline)  → PostgreSQL + Redis + Kafka (full production stack)
Phase 3+ (Production Scale)     → External caches (CDN), additional Kafka consumers (alerting, SIEM)
Phase 6 (Searchable Audit)      → Elasticsearch/OpenSearch (see PRD-006 FR-3)
```

### Migration Path

All stores implement the same interface, so migration is DI wiring only:

```go
// Phase 0-1: In-memory (development/testing)
userStore := inmemory.NewUserStore()

// Phase 2+: PostgreSQL (production)
userStore := postgres.NewUserStore(db)
```

**No business logic changes required.** Services depend on interfaces, not implementations.

### Storage Decision Matrix

| Store | Stay In-Memory When | Migrate to PostgreSQL When | Migrate to Redis When |
|-------|--------------------|-----------------------------|----------------------|
| `UserStore` | Single instance, ephemeral users | Users must persist across deploys | Never (not a cache) |
| `SessionStore` | Single instance, no DR needs | Never (sessions are transient) | Multi-instance deployment |
| `ConsentStore` | Development/testing only | GDPR compliance, audit requirements | Consent projections (CQRS read model) |
| `AuditStore` | Development/testing only | Any production deployment | Never (requires durability) |
| `RateLimitStore` | Single instance | Never | Multi-instance deployment |
| `RegistryCache` | Small dataset, testing | Never (it's a cache) | Large dataset, multi-instance |

### Event Streaming Decision Matrix

| Component | Stay Synchronous When | Migrate to Kafka When |
|-----------|----------------------|----------------------|
| `AuditPublisher` | Development/testing only, single instance | Production deployment; decouples audit writes from request path (PRD-006 TR-5) |
| `ConsentEvents` | Simple projections, low volume | CQRS read model updates require reliable delivery |
| `EventNotifications` | Low-volume systems, testing | Multi-consumer needs (audit, alerting, SIEM export) |

**Kafka Configuration Baseline (Phase 2):**

| Setting | Value | Rationale |
|---------|-------|-----------|
| `acks` | `all` | Durability for audit events |
| `replication.factor` | `3` | Fault tolerance (production) |
| `min.insync.replicas` | `2` | Write availability during broker failure |
| `retention.ms` | `604800000` (7 days) | Sufficient replay window for consumer catch-up |

**Topics:**

| Topic | Purpose | Consumers |
|-------|---------|-----------|
| `credo.audit.events` | All audit events (outbox-published) | Audit store writer, (Phase 6: ES indexer) |
| `credo.consent.events` | Consent state changes | Projection updaters |

### This PRD's Role

PRD-020 marks the transition point. The health checks (`/health/ready`) defined here validate database, Redis, and Kafka connectivity, signaling that production infrastructure is now required.

---

## 1C. Event Infrastructure Transition

### Audit Event Streaming Architecture

Phase 2 introduces Kafka to decouple audit event publishing from the request path:

```
Request Handler
     │
     ▼
┌─────────────────┐     ┌─────────────┐     ┌─────────────────┐
│ Audit Publisher │────▶│ Outbox Table│────▶│ Outbox Worker   │
└─────────────────┘     │ (PostgreSQL)│     │ (Kafka Producer)│
                        └─────────────┘     └────────┬────────┘
                                                     │
                                                     ▼
                                            ┌─────────────────┐
                                            │ Kafka Topic     │
                                            │ credo.audit.*   │
                                            └────────┬────────┘
                                                     │
                                    ┌────────────────┼────────────────┐
                                    ▼                ▼                ▼
                            ┌───────────┐   ┌───────────────┐  ┌───────────────┐
                            │Audit Store│   │ ES Indexer    │  │ SIEM Export   │
                            │ (primary) │   │ (Phase 6)     │  │ (Phase 6)     │
                            └───────────┘   └───────────────┘  └───────────────┘
```

### Outbox Pattern (from PRD-006 TR-5)

The outbox pattern ensures guaranteed delivery:

1. **Transactional Write:** Handler writes audit event to `audit_outbox` table within the same transaction as the business operation
2. **Polling Worker:** Background worker polls outbox, publishes to Kafka, marks events as published
3. **Idempotency:** Kafka consumers use event ID for deduplication
4. **Fallback:** If Kafka unavailable, outbox accumulates; events replay when Kafka recovers

### Graceful Degradation

| Kafka State | Behavior |
|-------------|----------|
| **Healthy** | Outbox worker publishes events within 100ms of creation |
| **Degraded (slow)** | Outbox queue depth increases; alert at 1000 pending events |
| **Down** | Events accumulate in outbox; no data loss; replay on recovery |
| **Partitioned** | Producer retries with exponential backoff; local buffering |

### When NOT to Use Kafka (Phase 2)

| Use Case | Recommendation |
|----------|----------------|
| Synchronous audit queries (`/me/data-export`) | Read from PostgreSQL audit store directly |
| Low-latency consent lookups | Redis projections, not Kafka |
| Real-time dashboards | Wait for Phase 6 (ES indexer) |

---

## 1D. Docker Compose Development Stack

Phase 2 requires a local development stack with PostgreSQL, Redis, and Kafka (Redpanda).

### Services

```yaml
services:
  # PostgreSQL - Primary database
  postgres:
    image: postgres:latest
    container_name: credo-postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: credo
      POSTGRES_PASSWORD: credo_dev_password
      POSTGRES_DB: credo
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U credo"]
      interval: 10s
      timeout: 5s
      retries: 5

  # pgAdmin - PostgreSQL Web Console
  pgadmin:
    image: dpage/pgadmin4:latest
    container_name: credo-pgadmin
    ports:
      - "5050:80"
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@credo.local
      PGADMIN_DEFAULT_PASSWORD: admin
    depends_on:
      - postgres

  # Redis - Sessions and rate limiting
  redis:
    image: redis:latest
    container_name: credo-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redpanda - Kafka-compatible broker
  redpanda:
    image: redpandadata/redpanda:latest
    container_name: credo-redpanda
    command:
      - redpanda start
      - --smp 1
      - --memory 1G
      - --overprovisioned
      - --node-id 0
      - --kafka-addr PLAINTEXT://0.0.0.0:9092
      - --advertise-kafka-addr PLAINTEXT://redpanda:9092
    ports:
      - "9092:9092"
    healthcheck:
      test: ["CMD", "rpk", "cluster", "health"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redpanda Console - Kafka Web UI
  redpanda-console:
    image: redpandadata/console:latest
    container_name: credo-redpanda-console
    ports:
      - "8085:8080"
    environment:
      KAFKA_BROKERS: redpanda:9092
    depends_on:
      - redpanda

volumes:
  postgres-data:
```

### Console Access

| Service | URL | Credentials | Purpose |
|---------|-----|-------------|---------|
| **pgAdmin** | http://localhost:5050 | admin@credo.local / admin | PostgreSQL admin (queries, tables, schema) |
| **Redpanda Console** | http://localhost:8085 | None | Kafka topics, messages, consumer groups |

### Backend Environment Variables

```bash
DATABASE_URL=postgres://credo:credo_dev_password@postgres:5432/credo?sslmode=disable
REDIS_URL=redis://redis:6379
KAFKA_BROKERS=redpanda:9092
```

---

## 1E. Integration with PRD-006 (Audit & Compliance)

PRD-020's Kafka introduction in Phase 2 directly supports PRD-006 TR-5 (Event Streaming & Indexing Pipeline):

| PRD-006 Requirement | PRD-020 Support |
|---------------------|-----------------|
| "Publish audit events to Kafka/NATS topics" | Kafka introduced in Phase 2; `credo.audit.events` topic |
| "Keep synchronous store append as fallback" | PostgreSQL audit store remains primary; outbox pattern ensures durability |
| "Outbox pattern guarantees delivery into Kafka" | Outbox table in PostgreSQL; polling worker publishes to Kafka |
| "Idempotent projection/indexers using event IDs" | Event ID included in Kafka message key for consumer deduplication |

**Phase 6 Continuation:**

The Elasticsearch indexer and searchable audit queries (PRD-006 FR-3) consume from the same Kafka topic established in Phase 2. This enables:

- Zero changes to event producers when adding new consumers
- Replay capability for index rebuilds
- Decoupled scaling of indexing from core API latency

---

## 2. Functional Requirements

### FR-0: Metrics and Alerting Baseline

**Required metrics:**
- Auth SLIs: p95/p99 latency and error rate for `/auth/authorize`, `/auth/token`, `/auth/revoke`, split by tenant and client.
- Revocation health: TRL write failures, revocation lag, and revoked-token check failures.
- Audit durability: enqueue depth, drop count, persist failures, and time-to-persist.
- Abuse signals: refresh token reuse detections, auth lockouts, and rate-limit denials by IP and client.
- Consent enforcement: consent gating failures and regulated-mode PII minimization violations.

**Required alerts:**
- Sustained auth SLI violations by tenant or client.
- TRL write or check failure spikes; revocation lag above threshold.
- Audit event drops or persist failures above threshold.
- Refresh token reuse spike or auth lockout surge.
- Consent gating failures above baseline.

### FR-1: Health Check Endpoints

**Endpoint:** `GET /health/live`

**Purpose:** Kubernetes liveness probe (is process alive?)

**Response (200):**

```json
{
  "status": "ok",
  "timestamp": "2025-12-12T10:00:00Z"
}
```

**Logic:** Return 200 if server is running, 500 if panicking

---

**Endpoint:** `GET /health/ready`

**Purpose:** Kubernetes readiness probe (can accept traffic?)

**Response (200):**

```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "registry_api": "ok"
  },
  "timestamp": "2025-12-12T10:00:00Z"
}
```

**Response (503 if not ready):**

```json
{
  "status": "not_ready",
  "checks": {
    "database": "ok",
    "redis": "failed",
    "kafka": "ok",
    "registry_api": "ok"
  },
  "timestamp": "2025-12-12T10:00:00Z"
}
```

**Logic:**

- Check database connection
- Check Redis connection
- Check Kafka broker connectivity
- Check external API reachability
- Return 503 if any check fails

**Kafka Health Check:**

| Check | Method | Failure Impact |
|-------|--------|----------------|
| Broker reachable | TCP connect to bootstrap servers | Service not ready (503) |
| Producer initialized | `producer.Flush(0)` returns no error | Service not ready (503) |
| Topic exists | Admin API `DescribeTopics` | Warning only (service ready but degraded) |

**Provider Health Checks** (identified gap from module README):

The `/health/ready` endpoint should include health checks for registered registry providers. Each provider exposes a `Health()` method through the provider interface that returns availability status.

- Wire provider `Health()` methods to the readiness probe
- Include per-provider health status in the `checks` response (e.g., `"citizen_provider": "ok"`, `"sanctions_provider": "degraded"`)
- Support configurable provider health thresholds (e.g., mark ready if 2-of-3 providers are healthy)
- Emit metrics for provider health state changes
- Log provider health check failures with circuit breaker context

**Example Response with Provider Checks:**

```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "providers": {
      "citizen": "ok",
      "sanctions": "ok",
      "biometric": "degraded"
    }
  },
  "timestamp": "2025-12-27T10:00:00Z"
}
```

---

**Endpoint:** `GET /health`

**Purpose:** General health status

**Response:**

```json
{
  "status": "healthy",
  "version": "1.2.3",
  "uptime_seconds": 86400,
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "registry": "ok"
  }
}
```

---

### FR-2: Backup & Restore

**Database Backup:**

```bash
# Daily automated backup
pg_dump -h localhost -U credo credo_db > backup_$(date +%Y%m%d).sql

# Upload to S3
aws s3 cp backup_$(date +%Y%m%d).sql s3://credo-backups/
```

**Backup Schedule:**

- **Daily:** Full database backup (retained 30 days)
- **Hourly:** Incremental logs (retained 7 days)
- **Weekly:** Full snapshot (retained 90 days)

**Restore Procedure:**

```bash
# Download from S3
aws s3 cp s3://credo-backups/backup_20251212.sql .

# Restore
psql -h localhost -U credo credo_db < backup_20251212.sql
```

**Encryption:** All backups encrypted at rest (AES-256)

---

### FR-3: Disaster Recovery Plan

**Recovery Time Objective (RTO):** 4 hours
**Recovery Point Objective (RPO):** 1 hour

**DR Scenarios:**

| Scenario             | Likelihood | Impact   | Recovery Steps                                |
| -------------------- | ---------- | -------- | --------------------------------------------- |
| **Database failure** | Medium     | High     | Restore from latest backup, replay logs       |
| **Redis failure**    | Medium     | Medium   | Rebuild cache from database, gradual recovery |
| **Kafka failure**    | Medium     | Medium   | Outbox accumulates; replay on recovery; no data loss |
| **Region outage**    | Low        | Critical | Failover to DR region, DNS update             |
| **Data corruption**  | Low        | High     | Point-in-time recovery from backup            |

**DR Testing:** Quarterly DR drills

---

### FR-4: Incident Response Runbooks

**Location:** `docs/runbooks/`

**Runbooks:**

1. **High Error Rate**

   - Check `/metrics` for error codes
   - Check logs for stack traces
   - Scale up if CPU/memory constrained
   - Rollback recent deployment if regression

2. **Database Connection Pool Exhaustion**

   - Check active connections: `SELECT count(*) FROM pg_stat_activity;`
   - Kill long-running queries
   - Increase pool size temporarily
   - Investigate slow queries

3. **Rate Limit Exceeded Alerts**

   - Check top IPs from rate limit logs
   - Confirm legitimate vs attack traffic
   - Add attacker IPs to blocklist
   - Scale rate limiter if needed

4. **Registry API Down**
   - Check registry API health
   - Enable graceful degradation (cached responses)
   - Notify users of degraded service
   - Contact registry provider

5. **Kafka/Redpanda Down**
   - Check broker health via Redpanda Console (http://localhost:8085)
   - Verify outbox depth in PostgreSQL: `SELECT COUNT(*) FROM audit_outbox WHERE published = false;`
   - If outbox depth > 10000, investigate urgently
   - Restart broker if hung
   - Events will auto-replay when Kafka recovers (no data loss)

---

### FR-5: Performance SLAs

**Latency Targets:**

| Endpoint Class | p50     | p95     | p99     | Timeout |
| -------------- | ------- | ------- | ------- | ------- |
| **Auth**       | < 50ms  | < 100ms | < 200ms | 5s      |
| **Consent**    | < 30ms  | < 80ms  | < 150ms | 3s      |
| **Registry**   | < 200ms | < 500ms | < 1s    | 10s     |
| **Decision**   | < 100ms | < 300ms | < 800ms | 5s      |
| **Audit**      | < 20ms  | < 50ms  | < 100ms | 2s      |

**Availability Target:** 99.9% uptime (43 minutes downtime/month)

**Error Budget:** 0.1% (allows ~260 req failures per 1M requests)

---

### FR-6: Capacity Planning

**Current Capacity:**

- 2 instances x 2 CPU x 4GB RAM
- Database: 100GB storage
- Redis: 8GB memory

**Growth Projections:**

| Metric            | Current | 6mo  | 12mo  |
| ----------------- | ------- | ---- | ----- |
| **Users**         | 10K     | 50K  | 200K  |
| **Requests/day**  | 1M      | 5M   | 20M   |
| **Database size** | 10GB    | 50GB | 200GB |
| **Instances**     | 2       | 5    | 10    |

**Scaling Triggers:**

- CPU > 70% sustained → scale up
- Memory > 80% sustained → scale up
- Request latency p95 > 2x target → scale up

---

## 3. Implementation Steps

### Phase 1: Infrastructure Stack

1. Add PostgreSQL, Redis, Kafka (Redpanda) to docker-compose.yml
2. Add pgAdmin and Redpanda Console for admin UIs
3. Create database migrations for core tables (users, consents, audit, outbox)
4. Wire PostgreSQL stores (users, consents, audit)
5. Wire Redis stores (sessions, rate limiting)
6. Implement audit outbox table and polling worker
7. Configure Kafka producer for audit events

### Phase 2: Health Checks

1. Implement `/health/live` endpoint
2. Implement `/health/ready` with database, Redis, Kafka checks
3. Configure Kubernetes probes

### Phase 3: Backup & DR

1. Create backup scripts (pg_dump, Redis RDB)
2. Configure S3 bucket with lifecycle policies
3. Document Kafka topic retention policies
4. Test restore procedure
5. Document DR plan

### Phase 4: Runbooks

1. Write incident response runbooks (including Kafka/outbox)
2. Create on-call rotation
3. Set up PagerDuty/Opsgenie
4. Conduct DR drill

---

## 4. Acceptance Criteria

**Infrastructure Stack:**
- [ ] PostgreSQL running in docker-compose with health check
- [ ] Redis running in docker-compose with health check
- [ ] Kafka (Redpanda) running in docker-compose with health check
- [ ] pgAdmin accessible at http://localhost:5050
- [ ] Redpanda Console accessible at http://localhost:8085
- [ ] Database migrations create core tables (users, consents, audit, audit_outbox)

**Event Streaming:**
- [ ] Audit events published to `credo.audit.events` topic
- [ ] Outbox pattern implemented (PostgreSQL outbox table + polling worker)
- [ ] Outbox worker handles Kafka failures gracefully (accumulates, retries)
- [ ] Kafka consumer uses event ID for idempotent processing

**Health Checks:**
- [ ] `/health/live` returns 200 when server running
- [ ] `/health/ready` returns 503 when database down
- [ ] `/health/ready` returns 503 when Kafka unreachable
- [ ] Kubernetes probes configured in deployment.yaml

**Backup & DR:**
- [ ] Daily backups running and uploaded to S3
- [ ] Restore tested successfully
- [ ] DR plan documented with RTO/RPO

**Operations:**
- [ ] 5+ runbooks created (including Kafka/outbox runbook)
- [ ] Performance SLAs documented
- [ ] Capacity planning spreadsheet created
- [ ] On-call rotation established
- [ ] Kafka alerts configured (outbox depth, broker health)

---

## 5. Monitoring Alerts

**Critical Alerts (Page immediately):**

- Service down (all instances)
- Error rate > 5%
- p99 latency > 5s
- Database connection failures
- Kafka broker unreachable > 1 minute
- Audit outbox depth > 10000 pending events

**Warning Alerts (Slack notification):**

- Error rate > 1%
- p95 latency > 2x target
- CPU > 70%
- Memory > 80%
- Audit outbox depth > 1000 pending events
- Kafka consumer lag > 5 minutes
- Kafka producer errors > 10/minute

---

## Revision History

| Version | Date       | Author       | Changes                                                                                          |
| ------- | ---------- | ------------ | ------------------------------------------------------------------------------------------------ |
| 1.3     | 2026-01-01 | Engineering  | Added Kafka to Phase 2; Sections 1C (Event Infrastructure), 1D (Docker Compose), 1E (PRD-006 alignment); updated health checks, DR, alerts, acceptance criteria for full production stack |
| 1.2     | 2025-12-27 | Engineering  | Added provider health checks to FR-1 (identified gap from module README)                          |
| 1.1     | 2025-12-21 | Engineering  | Added Section 1B: Storage Infrastructure Transition (in-memory first philosophy, migration triggers, decision matrix) |
| 1.0     | 2025-12-12 | Product Team | Initial PRD                                                                                      |
