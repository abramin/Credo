# ID Gateway

Identity verification gateway built as a modular monolith. It simulates OIDC-style auth, consent, registry evidence, VC issuance/verification, decisions, and audit logging.

## What’s inside
- Platform: config loader, logger, HTTP server setup.
- Auth: users and sessions.
- Consent: purpose-based consent lifecycle.
- Evidence: registry lookups (citizen/sanctions) and verifiable credentials.
- Decision: rules engine that evaluates identity, sanctions, and VC signals.
- Audit: publisher/worker with append-only storage.
- Transport: HTTP router/handlers that delegate to the services.

## Documentation
- Architecture overview: `docs/architecture.md`
- Product requirements: `docs/prd/README.md` (links to PRDs for auth, consent, registry, VC, decision, audit, and user data rights).

## Run it
- `make dev` (hot reload if available) or `go run ./cmd/server`
