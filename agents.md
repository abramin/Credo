Got it. Here’s the **updated Cursor backend scaffold prompt** with the privacy, consent, audit, and regulated-mode features baked in. It stays concise but complete enough for Cursor to generate a proper first scaffold.

If you want, next step can be the fully generated folder structure and the initial code.

---

# Cursor Prompt (Backend Scaffold with Regulated-Mode Features)

**SYSTEM PURPOSE**
Generate the backend for a small identity-verification gateway.
The service handles OIDC-style login, purpose-based consent, registry checks, verifiable credential issuance, and a decision engine.
It must include built-in demonstrations of regulated-domain concerns: privacy, consent, minimal data exposure, retention, and auditability.

---

# Core Architecture

Build a Go backend with these layers:

### 1. `/cmd/server`

• Main entry, config loading, DI setup.
• Support `REGULATED_MODE=true` environment variable.
• When regulated mode is on: stricter logging, forced consent, data minimisation, retention TTL enforcement.

### 2. `/internal/http`

• Handlers only, no logic.
• Routes:

* POST `/auth/authorize`
* POST `/auth/consent`
* POST `/auth/token`
* GET `/auth/userinfo`
* POST `/vc/issue`
* POST `/vc/verify`
* POST `/registry/citizen`
* POST `/registry/sanctions`
* POST `/decision/evaluate`
* GET `/me/data-export`
* DELETE `/me`
  • Automatic mapping of domain errors to HTTP responses.
  • JSON only.

### 3. `/internal/domain`

Pure business logic.
Include:
• OIDC mock flows
• Consent model: purpose-based, timestamped, revocable, required before registry or VC operations
• Data minimisation transformations (e.g. full DOB → derived `isOver18`)
• VC lifecycle logic
• Registry-check orchestration
• Decision engine (pass, pass_with_conditions, fail)
• Data retention policies
• Audit events emitted here, not at HTTP layer.

### 4. `/internal/consent`

Explicit consent objects and services:
• `ConsentRecord` with purpose, granted_at, expires_at, revoked_at.
• Enforce purpose binding.
• Evaluate whether a given operation requires consent.

### 5. `/internal/audit`

• Append-only audit service.
• Structured entries: userID, action, purpose, timestamp, requesting client, data subject, decision+reason.
• In-memory sink first, interface-based.

### 6. `/internal/storage`

Interface-driven stores:
• UserStore
• SessionStore
• ConsentStore
• VCStore
• RegistryCacheStore
• AuditStore
Each with a basic in-memory implementation.

### 7. `/internal/registry`

Mocked registries:
• `CitizenRegistryClient` returns deterministic record containing PII.
• `SanctionsRegistryClient` returns PEP/sanctions flags.
Include artificial latency.
Include optional “restricted data fields” removed when regulated mode is on.

### 8. `/internal/oidc`

Minimal OIDC mock:
• authorize
• consent
• token
• userinfo
Keep this simple and clear.

### 9. `/internal/vc`

Simple JSON-LD style VC objects.
Include mock signature + mock verification.
Support revocation.

### 10. `/internal/decision`

Combine identity + registry outputs + VC claims.
Return: `pass` | `pass_with_conditions` | `fail`.

### 11. `/internal/policy`

Retention windows (e.g. delete registry data after X minutes).
Data classification comments or tags on structs.

### 12. `/pkg/errors`

Typed errors (InvalidConsent, MissingConsent, PolicyViolation, RegistryTimeout, etc).

### 13. `/pkg/testutil`

BDD helpers.

### 14. `/test`

Include clear Given-When-Then tests such as:
• Given a user logs in, When they approve consent for future verification, Then consent is recorded.
• Given regulated mode is on, When calling citizen registry, Then only minimal fields are stored.
• Given user is verified, When requesting VC issuance, Then a VC is issued and an audit log entry created.
• etc.

---

# Regulated-Mode Features to Implement

1. **Purpose-Based Consent Enforcement**
   Operations (registry, VC issue, decision) require explicit consent with matching purpose.

2. **Data Minimisation**
   When `REGULATED_MODE=true`:
   • citizen registry returns only needed fields
   • domain layer strips identifiers before decision engine
   • VC issuance uses derived attributes instead of raw PII

3. **Audit Logging**
   Emit structured audit logs for:
   • login
   • consent granted/revoked
   • registry queries
   • VC issuance/verification
   • decisions
   • data export/deletion

4. **Data Retention**
   In-memory TTL deletion for sensitive registry data.
   Document retention windows in code comments.

5. **User Rights**
   • GET `/me/data-export` returns all data tied to the subject
   • DELETE `/me` deletes user records, sessions, consent, VCs

6. **Clear Struct Classification**
   Add comments or struct tags to highlight PII and derived fields.

---

# Clean Code Expectations

• Functions should do one thing and have names that reflect intent.
• Domain logic must be pure and testable.
• No business rules in HTTP handlers.
• No circular dependencies.
• Small, explicit interfaces.
• BDD-style test naming everywhere.
* Use gomock for test mocks
* Use sqlc for any sql 
