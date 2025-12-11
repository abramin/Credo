# Consent Module

**Purpose:** Manages purpose-based consent following GDPR Article 7 requirements.

---

## Architecture

This module follows **hexagonal architecture** (ports-and-adapters) pattern:

```
┌─────────────────────────────────────────────────────────┐
│                    External Layer                        │
│  ┌───────────────────────────────────────────────────┐  │
│  │  HTTP Handler (transport/http/handlers_consent.go)│  │
│  │  - POST /auth/consent                             │  │
│  │  - POST /auth/consent/revoke                      │  │
│  │  - GET /auth/consent                              │  │
│  └────────────────────┬──────────────────────────────┘  │
└───────────────────────┼─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                  Domain Layer (Core)                     │
│  ┌───────────────────────────────────────────────────┐  │
│  │  consent.Service                                  │  │
│  │  - Grant(ctx, userID, purposes)                   │  │
│  │  - Revoke(ctx, userID, purposes)                  │  │
│  │  - Require(ctx, userID, purpose) -> error         │  │
│  │  - List(ctx, userID, filter)                      │  │
│  └───────────────┬───────────────────────────────────┘  │
│                  │                                        │
│                  │ Depends on                             │
│                  ▼                                        │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Ports (Interfaces)                               │  │
│  │  - Store interface                                │  │
│  │  - audit.Publisher interface                      │  │
│  └───────────────────────────────────────────────────┘  │
└──────────────────────┬──────────────────────────────────┘
                       │
                       │ Implemented by
                       ▼
┌─────────────────────────────────────────────────────────┐
│               Infrastructure Layer                       │
│  ┌──────────────────┐    ┌──────────────────────────┐   │
│  │ Adapters         │    │  Storage                 │   │
│  │  - gRPC Server   │    │  - InMemoryStore         │   │
│  │    (inbound)     │    │  - PostgresStore (V2)    │   │
│  └──────────────────┘    └──────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## Layers

### 1. Domain Layer (Core Business Logic)

**Location:** `service.go`, `models.go`

**Responsibilities:**
- Enforce consent lifecycle (grant, revoke, expire)
- Validate purposes
- Apply idempotency rules (5-minute window)
- Emit audit events

**No dependencies on:**
- gRPC / Protobuf
- HTTP / JSON
- Database implementation

### 2. Ports (Interfaces)

**Location:** `store.go`

**Interfaces:**
- `Store` - Consent persistence
- `audit.Publisher` - Audit event emission

### 3. Adapters (Infrastructure)

#### HTTP Adapter (Outbound - to clients)
**Location:** `internal/transport/http/handlers_consent.go`
- Translates HTTP/JSON → Domain calls
- Maps domain errors → HTTP status codes
- Extracts user from JWT context

#### gRPC Server Adapter (Inbound - from other services)
**Location:** `adapters/grpc/server.go`
- Exposes consent service over gRPC
- Implements `consentpb.ConsentServiceServer`
- Translates Protobuf ↔ Domain models
- Used by: registry, decision, vc services

**Example gRPC call (from registry service):**

```go
// Registry service calls consent via gRPC
consentClient := grpc.NewConsentClient("localhost:9091")
err := consentClient.RequireConsent(ctx, userID, "registry_check")
if err != nil {
    return errors.New("missing consent")
}
```

#### Storage Adapter
**Location:** `store_memory.go` (V1), `store_postgres.go` (V2)
- Implements `Store` interface
- Handles persistence details

---

## Interservice Communication

### Consumed by (Inbound gRPC Calls)

Other services call consent service via gRPC:

1. **Registry Service** → Check consent before citizen/sanctions lookup
2. **VC Service** → Check consent before issuing credentials
3. **Decision Service** → Check consent before evaluation
4. **Biometric Service** → Check consent before face matching

### Provides (gRPC API)

**Service:** `ConsentService` (defined in `api/proto/consent.proto`)

**Methods:**
- `HasConsent(userID, purpose) → bool`
- `RequireConsent(userID, purpose) → error`
- `GrantConsent(userID, purposes[]) → ConsentRecord[]`
- `RevokeConsent(userID, purposes[]) → ConsentRecord[]`
- `ListConsents(userID) → ConsentRecord[]`

---

## Key Concepts

### Purpose-Based Consent

Consent is granted per-purpose, not globally:

```go
const (
    PurposeLogin                = "login"
    PurposeRegistryCheck        = "registry_check"
    PurposeVCIssuance           = "vc_issuance"
    PurposeDecisionEvaluation   = "decision_evaluation"
    PurposeBiometricVerification = "biometric_verification"
)
```

### Consent Lifecycle

```
[No Consent] --Grant--> [Active] --Expire--> [Expired]
                          |
                          +-------Revoke----> [Revoked]
```

### Idempotency

Repeated grant requests within 5 minutes return existing consent without:
- Updating timestamps
- Emitting audit events

This prevents audit noise from double-clicks/retries.

### Consent ID Reuse

One consent ID per user+purpose combination:
- Active consent: reuse ID, extend TTL
- Expired consent: reuse ID, renew
- Revoked consent: reuse ID, clear RevokedAt

---

## Testing

### Unit Tests

```go
func TestService_Grant(t *testing.T) {
    store := NewInMemoryStore()
    auditor := audit.NewMockPublisher()
    service := NewService(store, auditor, 365*24*time.Hour)

    // Grant consent
    records, err := service.Grant(ctx, "user_123", []Purpose{"registry_check"})
    assert.NoError(t, err)
    assert.Len(t, records, 1)
}
```

### Integration Tests with gRPC

```go
func TestGRPCConsent_RequireConsent(t *testing.T) {
    // Start gRPC server
    server := grpc.NewConsentServer(consentService)
    lis, _ := net.Listen("tcp", ":0")
    grpcServer := grpc.NewServer()
    consentpb.RegisterConsentServiceServer(grpcServer, server)
    go grpcServer.Serve(lis)

    // Create client
    conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
    client := consentpb.NewConsentServiceClient(conn)

    // Call RequireConsent
    resp, err := client.RequireConsent(ctx, &consentpb.RequireConsentRequest{
        UserId: "user_123",
        Purpose: consentpb.Purpose_PURPOSE_REGISTRY_CHECK,
    })
    assert.NoError(t, err)
    assert.False(t, resp.Allowed) // No consent granted yet
}
```

### Mock Port for Testing

Use `gomock` to generate mocks:

```bash
mockgen -source=store.go -destination=mocks/mock_store.go -package=mocks
```

---

## Future Enhancements

- **V2:**
  - Per-purpose TTL configuration
  - Consent templates (bundles)
  - Consent delegation (parent → child)
  - Async background worker for expiry notifications

- **Microservices Migration:**
  - Extract consent service to separate process
  - Run gRPC server on port 9091
  - Other services connect via `consent-service:9091`
  - No code changes to domain logic

---

## References

- PRD: [PRD-002-Consent-Management.md](../../docs/prd/PRD-002-Consent-Management.md)
- API Contract: [api/proto/consent.proto](../../api/proto/consent.proto)
- Architecture: [docs/architecture.md](../../docs/architecture.md#consent)

---

## Revision History

| Version | Date       | Changes                                                |
| ------- | ---------- | ------------------------------------------------------ |
| 1.0     | 2025-12-03 | Initial README                                         |
| 2.0     | 2025-12-11 | Added hexagonal architecture, gRPC adapters, ports     |
