# PRD-004B: Enhanced Verifiable Credentials (BBS+ & Status Lists)

**Status:** Not Started
**Priority:** P1 (High - Privacy & Revocation)
**Owner:** Engineering Team
**Dependencies:** PRD-004 (Verifiable Credentials)
**Last Updated:** 2025-12-06

---

## 1. Overview

### Problem Statement
The baseline VC implementation (PRD-004) issues JSON/JWT credentials without selective disclosure or scalable revocation. Relying parties either learn more data than necessary (privacy leak) or cannot efficiently check revocation for large credential sets.

### Goals
- Add privacy-preserving selective disclosure using BBS+ signatures.
- Provide scalable revocation via W3C Status List 2021.
- Keep backward compatibility with existing VC issuance/verification APIs.
- Support migration path for previously issued credentials.
- Document cryptographic choices and operational trade-offs.

### Non-Goals
- Implementing anonymous credentials or unlinkability guarantees beyond BBS+.
- Ledger/blockchain anchoring of status lists.
- Multi-issuer trust federation (single issuer for MVP).
- Mobile wallet UX (server-side APIs only).

---

## 2. User Stories

**As an end user**
- I want to reveal only the claims required (e.g., age over 18) when presenting a credential.

**As a verifier**
- I want to validate selective disclosure proofs and revocation status with a single API call.

**As an issuer**
- I want to revoke credentials individually without reissuing the entire set.

**As an administrator**
- I want observability into revocation and proof verification to audit compliance.

---

## 3. Functional Requirements

### FR-1: BBS+ Issuance
- Support issuing BBS+ signed credentials for schemas defined in PRD-004 (at least Age Verification and KYC-lite).
- Provide issuer key generation and rotation; publish public keys at `/.well-known/vc-issuer-keys.json`.

### FR-2: Selective Disclosure Presentation
- Accept proof requests specifying required claims or predicates (e.g., `age >= 18`).
- Produce BBS+ zero-knowledge presentations containing only requested fields.
- Verify presentations server-side; return structured verification result and error codes.

### FR-3: Status List Revocation
- Maintain W3C Status List 2021 documents (bitstring) per credential type.
- Endpoint: `GET /vc/status-lists/{listId}` returns compressed status list with metadata (ttl, size, lastUpdated).
- Endpoint: `POST /vc/revoke` updates list; idempotent and auditable.

### FR-4: Backward Compatibility
- Continue supporting existing JWT/JSON credentials; document which features are BBS+-only.
- Provide migration guidance for issuers/verifiers (flags, version identifiers in credential metadata).

### FR-5: Verification API
- Endpoint: `POST /vc/verify` accepts credential or presentation, returns:
  - `is_valid` (bool), `revoked` (bool), `errors` (array), `credential_type`, `version`.
  - For selective disclosure, return `disclosed_claims` map.

### FR-6: Observability & Security
- Emit audit events for issuance, revocation, verification (with request ID, credential ID hash, outcome).
- Metrics: issuance count, revocation count, verification success/failure, status list fetch latency.
- Protect revocation endpoints with issuer auth (API key or mTLS for MVP).

---

## 4. Data & Cryptography
- **Signature scheme:** BBS+ over BLS12-381 (pairing-friendly curve). Use a maintained library; document version.
- **Credential format:** JSON-LD with BBS+ proof section; include `credentialStatus` pointing to status list.
- **Status list:** Bitstring stored compressed; support up to 100k entries per list for MVP.
- **Key management:** Separate signing key per credential type; rotation policy documented; include key ID in credentials.

---

## 5. Acceptance Criteria
- Issuance API returns BBS+ credential with embedded status list entry and key ID.
- Verification API accepts a selective disclosure presentation and returns `is_valid=true` when proof and status are correct; returns `revoked=true` for revoked IDs.
- Status list endpoint serves compressed list within 200ms for 100k entries.
- Backward compatibility: legacy JWT VC verification still works; documentation calls out differences.
- Audit/metrics emitted for issuance, revocation, verification.

---

## 6. Risks & Open Questions
- Library maturity and security review for chosen BBS+ implementation.
- Status list distribution freshness (cache headers, CDN?).
- Storage size and rotation strategy for multiple lists.
- UX for predicate proofs beyond simple claim disclosure (future work).
