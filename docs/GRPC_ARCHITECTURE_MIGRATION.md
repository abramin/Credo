# gRPC + Hexagonal Architecture Migration Guide

**Date:** 2025-12-11
**Version:** 2.0
**Author:** Engineering Team

---

## Executive Summary

Credo has been upgraded to use **hexagonal architecture** (ports-and-adapters) with **gRPC for interservice communication**. This document outlines all changes made across code, documentation, and build systems.

### Key Changes

1. ✅ **Protobuf API Contracts** - Defined in `api/proto/`
2. ✅ **Port Interfaces** - Domain layer depends only on interfaces
3. ✅ **gRPC Adapters** - Client and server adapters for interservice calls
4. ✅ **Architecture Documentation** - Updated with hexagonal diagrams
5. ✅ **Build System** - Makefile targets for proto generation
6. ⏳ **PRD Updates** - Add interservice communication sections (pending)
7. ⏳ **CI/CD** - Proto validation in GitHub Actions (pending)

---

## Architecture Overview

### Before: Direct Service Dependencies

```
Registry Service
    │
    ├─> import "internal/consent"
    └─> consentService.Require(ctx, userID, purpose)
```

**Problems:**

- Tight coupling between modules
- Hard to test (requires full consent service)
- Cannot extract services to microservices
- Circular dependency risks

### After: Hexagonal Architecture with gRPC

```
Registry Service (Domain)
    │
    ├─> Depends on: ports.ConsentPort (interface)
    │
    └─> Injected: grpc.ConsentClient (adapter)
            │
            └─> gRPC call to Consent Service
```

**Benefits:**

- Loose coupling via interfaces
- Easy to mock for testing
- Ready for microservices migration
- Type-safe contracts (protobuf)

---

## Protobuf API Contracts

- Can be found in api/proto

---

## Port Interfaces (Domain Layer)

### ConsentPort (`internal/registry/ports/consent.go`)

**Key Points:**

- No gRPC/protobuf imports
- No HTTP/JSON imports
- Pure domain interface
- Easy to mock for testing

### RegistryPort (`internal/decision/ports/registry.go`)

---

## gRPC Adapters

### Server Adapter (Inbound)

**Responsibilities:**

- Validate protobuf requests
- Translate proto ↔ domain models
- Handle gRPC-specific errors
- Call domain service (no business logic)

### Client Adapter (Outbound)

**Location:** `internal/registry/adapters/grpc/consent_client.go`

**Responsibilities:**

- Implement port interface
- Add timeouts and metadata
- Translate domain ↔ proto models
- Map gRPC errors to domain errors
- Handle connection failures

---

## Dependency Injection (Wiring)

### Main.go Pattern

```go
package main

func main() {
	// 1. Create stores
	consentStore := consent.NewInMemoryStore()

	// 2. Create domain services
	consentService := consent.NewService(consentStore, auditor, ttl)

	// 3. Create gRPC server adapter
	consentGRPCServer := grpc.NewConsentServer(consentService)

	// 4. Start gRPC server
	lis, _ := net.Listen("tcp", ":9091")
	grpcServer := grpc.NewServer()
	consentpb.RegisterConsentServiceServer(grpcServer, consentGRPCServer)
	go grpcServer.Serve(lis)

	// 5. Create gRPC client adapter for other services
	consentClient, _ := grpc.NewConsentClient("localhost:9091", 5*time.Second)

	// 6. Inject client into registry service
	registryService := registry.NewService(
		store,
		consentClient, // <-- Implements ports.ConsentPort
	)

	// 7. HTTP handlers use same services
	httpHandler := transport.NewHandler(
		consentService,
		registryService,
		// ...
	)
}
```

---

## Build System Updates

### Makefile Targets

#### Generate Proto Files

```bash
make proto-gen
```

**What it does:**

- Runs `protoc` on all `.proto` files
- Generates `.pb.go` (message types) and `_grpc.pb.go` (service stubs)
- Output: `api/proto/common/commonpb/*.pb.go`, etc.

#### Check Proto Files

```bash
make proto-check
```

**What it does:**

- Verifies generated files match proto definitions
- Fails if proto files were modified but not regenerated
- Used in CI to prevent stale generated code

#### Clean Proto Files

```bash
make proto-clean
```

**What it does:**

- Removes all generated `.pb.go` files
- Useful before full rebuild

### Installation Requirements

```bash
# Install protoc compiler
brew install protobuf  # macOS
apt install protobuf-compiler  # Ubuntu

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Verify installation
protoc --version  # Should be 3.x or higher
which protoc-gen-go protoc-gen-go-grpc  # Should find both
```

---

## Migration Path to Microservices

### Phase 1: Monolith with in-process gRPC (Current)

```
┌────────────────────────────────────┐
│       Single Process (Port 8080)   │
│                                    │
│  ┌──────────┐   ┌──────────────┐  │
│  │ Consent  │   │  Registry    │  │
│  │ Service  │   │  Service     │  │
│  └────┬─────┘   └──────┬───────┘  │
│       │                │           │
│       └────────────────┘           │
│        In-memory calls             │
└────────────────────────────────────┘
```

### Phase 2: Extract Consent Service

```
┌─────────────────────┐      ┌──────────────────────┐
│  Consent Service    │      │  Gateway Service     │
│  (Port 9091)        │◄─────│  (Port 8080)         │
│                     │ gRPC │                      │
│  ┌──────────────┐   │      │  ┌────────────────┐  │
│  │  Consent     │   │      │  │  Registry      │  │
│  │  Service     │   │      │  │  Service       │  │
│  └──────────────┘   │      │  └────────────────┘  │
└─────────────────────┘      └──────────────────────┘
```

**Steps:**

1. Start consent service as separate process
2. Update `NewConsentClient("consent-service:9091")`
3. **No code changes to domain logic**
4. Deploy both services
5. Verify gRPC health checks

### Phase 3: Full Microservices

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   Auth       │   │  Consent     │   │  Registry    │
│   :9090      │   │  :9091       │   │  :9092       │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       └──────────────────┴──────────────────┘
                      gRPC
                       │
                       ▼
              ┌────────────────────┐
              │  API Gateway       │
              │  (HTTP → gRPC)     │
              └────────────────────┘
```

---

### Interservice Communication Model

**Internal API (gRPC):**

- Protocol: gRPC over HTTP/2
- Serialization: Protocol Buffers
- Auth: Metadata propagation (future: mTLS)
- Location: `api/proto/consent.proto`

**External API (HTTP):**

- Protocol: HTTP/1.1
- Serialization: JSON
- Auth: Bearer tokens (JWT)
- Location: `internal/transport/http/`

**Hexagonal Architecture:**

- Domain layer depends on port interfaces
- gRPC adapters implement ports
- Easy to swap implementations (gRPC, HTTP, in-memory)
- Ready for microservices migration

**Example (Registry → Consent):**

```go
// Domain layer (registry service)
type Service struct {
    consentPort ports.ConsentPort // <-- Interface, not concrete type
}

// Adapter layer (gRPC client)
type ConsentClient struct {
    client consentpb.ConsentServiceClient
}

func (c *ConsentClient) RequireConsent(ctx, userID, purpose) error {
    // Translate domain → proto
    resp, err := c.client.RequireConsent(ctx, &consentpb.RequireConsentRequest{...})
    // Translate proto → domain error
    return mapGRPCError(err)
}

// Wiring (main.go)
consentClient := grpc.NewConsentClient("localhost:9091")
registryService := registry.NewService(store, consentClient)
```

**Future Enhancements:**

- Async background workers (pub/sub)
- Event-driven orchestration (Kafka, NATS)
- Circuit breakers for external services
- Service mesh (Istio, Linkerd) for mTLS and observability

---

## Common Issues & Solutions

### Issue: `protoc: command not found`

**Solution:**

```bash
# macOS
brew install protobuf

# Ubuntu
sudo apt-get install protobuf-compiler

# Verify
protoc --version
```

### Issue: `protoc-gen-go: program not found`

**Solution:**

```bash
# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add to PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify
which protoc-gen-go protoc-gen-go-grpc
```

### Issue: Generated files import wrong packages

**Solution:**
Check `option go_package` in `.proto` files:

```protobuf
option go_package = "github.com/credo/gateway/api/proto/consent;consentpb";
```

### Issue: Circular imports between services

**Solution:**
Use port interfaces, not direct imports:

```go
// ❌ Bad: Direct import
import "internal/consent"
type Service struct {
    consent *consent.Service
}

// ✅ Good: Port interface
import "internal/registry/ports"
type Service struct {
    consentPort ports.ConsentPort
}
```

---

## Best Practices

### 1. Keep Domain Layer Clean

**❌ Don't:**

```go
import consentpb "api/proto/consent"

type Service struct {
    client consentpb.ConsentServiceClient
}
```

**✅ Do:**

```go
import "internal/registry/ports"

type Service struct {
    consentPort ports.ConsentPort
}
```

### 2. Translate at Adapter Boundary

**❌ Don't:** Use protobuf types in domain

```go
func (s *Service) Citizen(ctx, req *registrypb.CheckCitizenRequest) (*registrypb.CitizenRecord, error)
```

**✅ Do:** Use domain models

```go
func (s *Service) Citizen(ctx context.Context, nationalID string) (*models.CitizenRecord, error)
```

### 3. Handle Errors Properly

Map gRPC errors to domain errors:

```go
func mapGRPCError(err error) error {
    st, ok := status.FromError(err)
    if !ok {
        return errors.NewGatewayError(errors.CodeInternal, "internal error", err)
    }

    switch st.Code() {
    case codes.InvalidArgument:
        return errors.NewGatewayError(errors.CodeInvalidArgument, st.Message(), err)
    case codes.NotFound:
        return errors.NewGatewayError(errors.CodeNotFound, st.Message(), err)
    case codes.PermissionDenied:
        return errors.NewGatewayError(errors.CodeMissingConsent, st.Message(), err)
    default:
        return errors.NewGatewayError(errors.CodeInternal, st.Message(), err)
    }
}
```

### 4. Add Timeouts and Retries

```go
func (c *ConsentClient) RequireConsent(ctx, userID, purpose) error {
    // Add timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Add retry logic (exponential backoff)
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := c.client.RequireConsent(ctx, req)
        if err == nil {
            return nil
        }

        // Don't retry on client errors
        if st, ok := status.FromError(err); ok {
            if st.Code() == codes.InvalidArgument || st.Code() == codes.PermissionDenied {
                return mapGRPCError(err)
            }
        }

        lastErr = err
        time.Sleep(time.Duration(attempt*100) * time.Millisecond)
    }
    return lastErr
}
```

### 5. Propagate Metadata

```go
func (c *ConsentClient) addMetadata(ctx context.Context) context.Context {
    requestID, _ := ctx.Value("request_id").(string)

    md := metadata.Pairs(
        "request-id", requestID,
        "timestamp", time.Now().Format(time.RFC3339),
    )
    return metadata.NewOutgoingContext(ctx, md)
}
```

---

## Next Steps

### Immediate (V1)

- [x] Create protobuf definitions
- [x] Add port interfaces
- [x] Implement gRPC adapters
- [x] Update architecture documentation
- [x] Add Makefile targets
- [ ] Update PRD-001, PRD-002, PRD-003 with interservice communication sections
- [ ] Add CI/CD proto validation
- [ ] Generate gomock mocks for ports

### Short-term (V2)

- [ ] Add retry logic with exponential backoff
- [ ] Implement circuit breakers
- [ ] Add gRPC interceptors for logging/metrics
- [ ] Use mTLS for production
- [ ] Add service discovery (Consul, etcd)
- [ ] Implement health checks per service

### Long-term (Microservices)

- [ ] Extract consent service
- [ ] Extract registry service
- [ ] Extract auth service
- [ ] Add API gateway (gRPC-Web, Envoy)
- [ ] Implement service mesh (Istio)
- [ ] Add distributed tracing (Jaeger, Zipkin)

---

## References

- **Architecture:** [docs/architecture.md](architecture.md)
- **Protobuf Docs:** https://protobuf.dev/
- **gRPC Go Tutorial:** https://grpc.io/docs/languages/go/
- **Hexagonal Architecture:** https://alistair.cockburn.us/hexagonal-architecture/
- **Consent Module README:** [internal/consent/README.md](../internal/consent/README.md)

---

## Revision History

| Version | Date       | Author           | Changes                                               |
| ------- | ---------- | ---------------- | ----------------------------------------------------- |
| 1.0     | 2025-12-11 | Engineering Team | Initial gRPC + hexagonal architecture migration guide |
