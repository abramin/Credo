# PRD-010: Zero-Knowledge Proofs for Privacy-Preserving Verification

**Status:** Not Started
**Priority:** P3 (Low - Advanced Feature)
**Owner:** Engineering Team
**Last Updated:** 2025-12-06
**Dependencies:** PRD-004 (Verifiable Credentials), PRD-009 (DIDs - optional)

---

## 1. Overview

### Problem Statement
Current credential verification requires sharing full credential data. Users must reveal their exact age (e.g., "born 1985-03-15") to prove they're over 18, or their exact salary to prove income threshold. This violates the principle of **data minimization** and creates unnecessary privacy exposure. Users should be able to prove statements ("I am over 18", "My income exceeds $50k") without revealing the underlying data.

### Goals
- Implement zero-knowledge proof (ZKP) system for privacy-preserving verification
- Enable users to prove predicates without revealing credential values
- Support common use cases: age verification, income thresholds, credential possession
- Integrate with existing Verifiable Credentials infrastructure
- Provide proof generation and verification APIs
- Document cryptographic guarantees and limitations
- Maintain performance (proof generation <1s, verification <100ms)

### Non-Goals
- General-purpose ZK computation (zkSNARKs for arbitrary programs)
- Blockchain integration (ZK-rollups, privacy coins)
- ZK-based authentication (focus is on credential disclosure)
- Multi-party computation (MPC)
- Homomorphic encryption
- Anonymous credentials (BBS+ signatures) - future consideration
- Mobile SDK with ZK circuits (server-side initially)

---

## 2. User Stories

### As an End User
- I want to prove I'm over 18 without revealing my birthdate
- I want to prove my income exceeds a threshold without revealing exact salary
- I want to prove I have a credential without showing its contents
- I want to prove I live in a specific country without revealing my full address
- I want assurance that my data isn't leaked during verification

### As a Relying Party (Service Provider)
- I want to verify age claims without collecting birthdates (GDPR minimization)
- I want to verify income thresholds for lending decisions
- I want to verify credential possession without accessing sensitive data
- I want cryptographic proof that claims are valid
- I want to define custom predicates for my use case

### As a Compliance Officer
- I want to ensure we collect minimum necessary data (GDPR Article 5)
- I want cryptographic proof of verification without PII storage
- I want to audit what predicates were verified (not the underlying data)
- I want to demonstrate privacy-by-design architecture

---

## 3. Technical Design

### 3.1 Zero-Knowledge Proofs Overview

**What is a ZKP?**
A zero-knowledge proof allows a prover to convince a verifier that a statement is true without revealing any information beyond the truth of the statement.

**Example:**
- **Statement:** "My age is greater than 18"
- **Secret:** Birthdate: 1985-03-15
- **Proof:** Cryptographic proof that age > 18
- **Verification:** Verifier confirms proof is valid, learns nothing about actual birthdate

**ZK Properties:**
1. **Completeness:** If statement is true, honest prover convinces honest verifier
2. **Soundness:** If statement is false, no prover can convince verifier (except with negligible probability)
3. **Zero-Knowledge:** Verifier learns nothing except that statement is true

### 3.2 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Identity Gateway                         │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Zero-Knowledge Proof System                  │  │
│  │                                                      │  │
│  │  ┌────────────┐         ┌──────────────────────┐   │  │
│  │  │  Circuit   │────────▶│   Proof Generator    │   │  │
│  │  │  Registry  │         │   (Prover)           │   │  │
│  │  └────────────┘         └──────────────────────┘   │  │
│  │                                 │                   │  │
│  │                                 │ Generates proof   │  │
│  │                                 ▼                   │  │
│  │         ┌──────────────────────────────────┐       │  │
│  │         │    Proof Verification            │       │  │
│  │         │    (Verifier)                    │       │  │
│  │         └──────────────────────────────────┘       │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │       Credential Service Integration                 │  │
│  │  ┌────────────┐         ┌──────────────────────┐   │  │
│  │  │    VCs     │────────▶│   ZK Proof Request   │   │  │
│  │  │  Storage   │         │   Handler            │   │  │
│  │  └────────────┘         └──────────────────────┘   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
              ┌──────────────────────────────┐
              │   ZK Circuit Definitions     │
              │  ┌────────────────────────┐  │
              │  │ age_over_threshold.r1cs│  │
              │  │ numeric_range.r1cs     │  │
              │  │ set_membership.r1cs    │  │
              │  │ credential_exists.r1cs │  │
              │  └────────────────────────┘  │
              └──────────────────────────────┘
```

### 3.3 ZK System Selection

**Option 1: Bulletproofs (Recommended for MVP)**
- **Pros:** No trusted setup, efficient for range proofs, established
- **Cons:** Larger proof size than SNARKs, slower verification
- **Use Cases:** Age verification, income thresholds, numeric ranges
- **Library:** dalek-cryptography/bulletproofs (Rust, mature)

**Option 2: Groth16 (zkSNARKs)**
- **Pros:** Tiny proofs (~200 bytes), fast verification
- **Cons:** Requires trusted setup per circuit, complex
- **Use Cases:** All use cases, but overhead of trusted setup
- **Library:** iden3/circom + snarkjs

**Option 3: PLONK/Halo2**
- **Pros:** Universal trusted setup (one-time), flexible
- **Cons:** Newer, less tooling, more complex
- **Use Cases:** Production systems with diverse circuits
- **Library:** ZCash halo2

**Decision:** Start with **Bulletproofs** for range proofs (age, income), evaluate Groth16 for production if proof size matters.

### 3.4 Data Model

**ZK Proof Request**
```go
type ZKProofRequest struct {
    ID                string                 `json:"id"`
    TenantID          string                 `json:"tenant_id"`
    RequesterID       string                 `json:"requester_id"`    // Relying party
    SubjectID         string                 `json:"subject_id"`      // User or DID
    CircuitType       string                 `json:"circuit_type"`    // "age_over", "range", etc.
    Predicate         Predicate              `json:"predicate"`
    CredentialID      string                 `json:"credential_id"`   // Which VC to prove about
    Status            string                 `json:"status"`          // "pending", "generated", "verified"
    CreatedAt         time.Time              `json:"created_at"`
    ExpiresAt         time.Time              `json:"expires_at"`
    Metadata          map[string]interface{} `json:"metadata"`
}

type Predicate struct {
    Type       string                 `json:"type"`        // "greater_than", "less_than", "in_range", "in_set"
    Attribute  string                 `json:"attribute"`   // Credential field to prove about
    Value      interface{}            `json:"value"`       // Threshold or comparison value
    Parameters map[string]interface{} `json:"parameters"`  // Additional params
}
```

**ZK Proof**
```go
type ZKProof struct {
    ID            string    `json:"id"`
    RequestID     string    `json:"request_id"`
    CircuitType   string    `json:"circuit_type"`
    ProofData     []byte    `json:"proof_data"`      // Serialized proof
    PublicInputs  []string  `json:"public_inputs"`   // Values visible to verifier
    ProofSystem   string    `json:"proof_system"`    // "bulletproofs", "groth16"
    GeneratedAt   time.Time `json:"generated_at"`
    VerifiedAt    *time.Time `json:"verified_at,omitempty"`
    IsValid       *bool     `json:"is_valid,omitempty"`
}
```

**ZK Circuit Definition**
```go
type ZKCircuit struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    CircuitType     string            `json:"circuit_type"`
    Description     string            `json:"description"`
    ProofSystem     string            `json:"proof_system"`
    R1CSPath        string            `json:"r1cs_path"`        // Compiled circuit
    ProvingKeyPath  string            `json:"proving_key_path"`
    VerifyingKeyPath string           `json:"verifying_key_path"`
    PublicInputs    []string          `json:"public_inputs"`
    PrivateInputs   []string          `json:"private_inputs"`
    Version         string            `json:"version"`
    CreatedAt       time.Time         `json:"created_at"`
}
```

### 3.5 Supported Use Cases

#### 1. Age Verification (age_over_threshold)
```
Circuit: age_over_threshold
Private Inputs: birthdate (YYYY-MM-DD)
Public Inputs: threshold_age, current_date
Statement: birthdate implies age >= threshold_age
```

**Example:**
```json
{
  "circuit_type": "age_over_threshold",
  "predicate": {
    "type": "greater_than_or_equal",
    "attribute": "birthdate",
    "value": 18
  },
  "credential_id": "vc_birthdate_123"
}
```

**Proof Generation:**
```
1. Extract birthdate from credential: 1985-03-15
2. Calculate age: 2025 - 1985 = 40
3. Generate ZK proof that 40 >= 18
4. Proof reveals: "age >= 18" (TRUE)
5. Proof hides: actual birthdate, exact age
```

#### 2. Income Threshold (numeric_range)
```
Circuit: numeric_range
Private Inputs: actual_value
Public Inputs: min_threshold, max_threshold (optional)
Statement: min_threshold <= actual_value <= max_threshold
```

**Example:**
```json
{
  "circuit_type": "numeric_range",
  "predicate": {
    "type": "in_range",
    "attribute": "annual_income",
    "value": {"min": 50000, "max": null}
  },
  "credential_id": "vc_income_456"
}
```

#### 3. Set Membership (set_membership)
```
Circuit: set_membership
Private Inputs: value
Public Inputs: merkle_root_of_set
Statement: value exists in set represented by merkle_root
```

**Example:** Prove citizenship without revealing country
```json
{
  "circuit_type": "set_membership",
  "predicate": {
    "type": "in_set",
    "attribute": "country",
    "value": ["USA", "CAN", "MEX", "GBR", "FRA", "DEU"]
  },
  "credential_id": "vc_nationality_789"
}
```

#### 4. Credential Existence (credential_exists)
```
Circuit: credential_exists
Private Inputs: credential_data, credential_signature
Public Inputs: issuer_public_key
Statement: I possess a valid credential signed by issuer
```

### 3.6 API Design

#### Request ZK Proof
```http
POST /api/v1/zk/proof-requests
Authorization: Bearer {token}
Content-Type: application/json

{
  "subject_id": "user_123",
  "circuit_type": "age_over_threshold",
  "predicate": {
    "type": "greater_than_or_equal",
    "attribute": "birthdate",
    "value": 18
  },
  "credential_id": "vc_birthdate_123",
  "expires_in_minutes": 10
}
```

**Response:**
```json
{
  "proof_request_id": "zk_req_abc123",
  "status": "pending",
  "challenge": "da2b8f3c4e5f67890123456789abcdef",
  "expires_at": "2025-12-06T10:40:00Z",
  "proof_endpoint": "/api/v1/zk/proofs"
}
```

#### Generate ZK Proof (User Action)
```http
POST /api/v1/zk/proofs
Authorization: Bearer {user_token}
Content-Type: application/json

{
  "proof_request_id": "zk_req_abc123",
  "credential_id": "vc_birthdate_123",
  "consent": true
}
```

**Response:**
```json
{
  "proof_id": "zk_proof_xyz789",
  "proof_request_id": "zk_req_abc123",
  "circuit_type": "age_over_threshold",
  "proof_data": "base64_encoded_proof_bytes",
  "public_inputs": {
    "threshold_age": 18,
    "current_date": "2025-12-06",
    "result": true
  },
  "proof_system": "bulletproofs",
  "generated_at": "2025-12-06T10:30:15Z"
}
```

#### Verify ZK Proof
```http
POST /api/v1/zk/proofs/{proof_id}/verify
Authorization: Bearer {requester_token}
```

**Response:**
```json
{
  "proof_id": "zk_proof_xyz789",
  "is_valid": true,
  "verified_at": "2025-12-06T10:30:20Z",
  "statement": "age >= 18",
  "verifier_confidence": "cryptographic",
  "public_inputs": {
    "threshold_age": 18,
    "result": true
  }
}
```

#### List Available Circuits
```http
GET /api/v1/zk/circuits
```

**Response:**
```json
{
  "circuits": [
    {
      "circuit_type": "age_over_threshold",
      "name": "Age Over Threshold",
      "description": "Prove age exceeds a threshold without revealing birthdate",
      "supported_predicates": ["greater_than", "greater_than_or_equal"],
      "proof_system": "bulletproofs"
    },
    {
      "circuit_type": "numeric_range",
      "name": "Numeric Range Proof",
      "description": "Prove a number falls within a range",
      "supported_predicates": ["in_range", "greater_than", "less_than"],
      "proof_system": "bulletproofs"
    },
    {
      "circuit_type": "set_membership",
      "name": "Set Membership",
      "description": "Prove value belongs to a set without revealing which",
      "supported_predicates": ["in_set"],
      "proof_system": "bulletproofs"
    }
  ]
}
```

### 3.7 Implementation Stack

**ZK Libraries**
- **Rust:** dalek-cryptography/bulletproofs (range proofs)
- **Rust:** arkworks-rs (general-purpose ZK)
- **Go Binding:** cgo or gRPC service (Go → Rust)

**Circuit Compilation**
- **Bulletproofs:** Hand-coded circuits (simple for range proofs)
- **SNARKs (future):** circom for circuit definition, snarkjs for compilation

**Storage**
- Proof requests in PostgreSQL
- Generated proofs in PostgreSQL (small) or blob storage (large)
- Circuit definitions in filesystem
- Proving/verifying keys in encrypted storage

**Integration Approach**
```
┌──────────────────┐
│  Go Service      │
│  (Identity GW)   │
└────────┬─────────┘
         │ gRPC
         ▼
┌──────────────────┐
│  Rust Service    │
│  (ZK Prover)     │
│                  │
│  - Bulletproofs  │
│  - Circuit logic │
│  - Key mgmt      │
└──────────────────┘
```

### 3.8 Security Considerations

**Trusted Setup (for SNARKs)**
- If using Groth16: multi-party computation (MPC) ceremony
- Participants contribute randomness
- If even one participant is honest, setup is secure
- Document setup participants and process

**Proof Soundness**
- Use well-audited ZK libraries
- Verify all proofs before accepting claims
- Rate limit proof generation (prevent grinding attacks)
- Set proof expiry (proofs are single-use)

**Private Input Protection**
- Private inputs never leave user device (future: client-side proving)
- Server-side proving: inputs encrypted in transit, wiped after proof gen
- Audit log proof generation without logging private inputs

**Verifier Security**
- Verifier must use correct verifying key
- Protect verifying keys from tampering
- Verify proof format and public inputs

---

## 4. Implementation Plan

### Phase 1: Research & Design (Week 1-2)
- [ ] Study Bulletproofs paper and implementation
- [ ] Design age verification circuit
- [ ] Prototype range proof in Rust
- [ ] Define API contracts
- [ ] Create ZK circuit database schema

### Phase 2: MVP - Age Verification (Week 3-4)
- [ ] Implement age_over_threshold circuit in Rust
- [ ] Build Rust gRPC service for proving/verification
- [ ] Create Go client for Rust ZK service
- [ ] Implement /zk/proof-requests endpoint
- [ ] Implement /zk/proofs generation endpoint
- [ ] Implement /zk/proofs verification endpoint

### Phase 3: Additional Circuits (Week 5-6)
- [ ] Implement numeric_range circuit
- [ ] Implement set_membership circuit
- [ ] Implement credential_exists circuit
- [ ] Add circuit registry and discovery API
- [ ] Integration tests for all circuits

### Phase 4: Integration (Week 7-8)
- [ ] Integrate with Verifiable Credentials service
- [ ] Add ZK proof option to consent flow
- [ ] Update credential presentation to support ZK proofs
- [ ] Admin UI for managing ZK proof requests
- [ ] User UI for granting ZK proof consent

### Phase 5: Production Hardening (Week 9-10)
- [ ] Security audit (circuit logic, key management)
- [ ] Performance optimization (proof generation <1s)
- [ ] Error handling and edge cases
- [ ] Monitoring and alerting
- [ ] Documentation (user guide, API docs, cryptographic specs)

### Phase 6: Advanced Features (Future)
- [ ] Client-side proof generation (WASM)
- [ ] Groth16 implementation for smaller proofs
- [ ] BBS+ signatures for anonymous credentials
- [ ] Recursive proofs (proof of proof)
- [ ] zkEVM integration (future)

---

## 5. Testing Strategy

### Unit Tests
- Circuit correctness (honest prover)
- Circuit soundness (dishonest prover rejected)
- Serialization/deserialization
- API input validation

### Integration Tests
- End-to-end proof generation and verification
- Integration with credential service
- Consent flow with ZK proofs
- Error scenarios (invalid proofs, expired credentials)

### Cryptographic Tests
- Soundness: Invalid proofs must fail verification
- Completeness: Valid proofs must pass verification
- Zero-knowledge: Verifier learns only public outputs
- Performance: Proof generation and verification times

### Security Tests
- Malformed proof rejection
- Proof replay attacks
- Private input leakage tests
- Verifying key tampering detection

### Load Tests
- Concurrent proof generation
- Proof verification throughput
- Circuit loading performance
- Memory usage under load

---

## 6. Success Metrics

### Cryptographic Metrics
- **Proof Size:** <10 KB (Bulletproofs) or <1 KB (SNARKs)
- **Proof Generation Time:** <1 second
- **Verification Time:** <100ms
- **Zero False Negatives:** All valid proofs verify
- **Zero False Positives:** All invalid proofs rejected

### Business Metrics
- Adoption rate (% of verifications using ZK)
- Privacy-sensitive use case coverage
- Compliance officer confidence (qualitative)
- Reduction in PII storage

### Operational Metrics
- Circuit compilation time
- Proof request completion rate
- System uptime and availability

---

## 7. Privacy & Compliance

### GDPR Benefits
- **Data Minimization (Art. 5.1c):** ZK proofs collect minimum data
- **Purpose Limitation (Art. 5.1b):** Specific predicates, not general data
- **Privacy by Design (Art. 25):** Cryptographic privacy guarantees
- **Data Subject Rights:** Proof verification without PII storage

### Audit Trail
- Log proof requests (predicate, not private inputs)
- Log proof verification results
- Do NOT log private inputs (birthdate, income, etc.)
- Compliance reports show "what was verified" not "what the data was"

### Consent
- User must consent to proof generation
- Explain what is being proved and what is hidden
- Allow users to decline and use traditional verification

---

## 8. Documentation Requirements

### For Developers
- ZK proof system overview
- Circuit specifications
- API integration guide
- Code examples (proof request, generation, verification)
- Cryptographic primitives explanation

### For End Users
- "What are Zero-Knowledge Proofs?" explainer
- How ZK proofs protect your privacy
- Step-by-step proof generation guide
- FAQ: What can verifiers learn?

### For Compliance Officers
- Cryptographic guarantees
- Privacy properties
- GDPR compliance mapping
- Audit trail specifications

### For Administrators
- Circuit deployment guide
- Key management procedures
- Performance tuning
- Monitoring and troubleshooting

---

## 9. Open Questions

1. **Client-Side vs. Server-Side Proving:**
   - MVP: Server-side (simpler, faster to implement)
   - Future: Client-side (better privacy, user controls private keys)
   - Trade-off: Complexity vs. privacy

2. **Proof Caching:**
   - Can proofs be reused? (e.g., age verification valid for 30 days)
   - Or must each request generate a new proof?
   - Security implications of proof reuse?

3. **Circuit Upgrades:**
   - How to upgrade circuits without breaking existing integrations?
   - Versioning strategy?
   - Backward compatibility?

4. **Multi-Attribute Proofs:**
   - Can we prove multiple predicates in one proof? (e.g., "age > 18 AND income > $50k")
   - Performance implications?
   - Circuit complexity?

5. **Interoperability:**
   - Can proofs generated by other systems be verified?
   - Standard proof formats?

---

## 10. Future Enhancements

- **Anonymous Credentials:** BBS+ signatures for unlinkable credentials
- **Recursive Proofs:** Proof composition (prove proof of proof)
- **zkEVM Integration:** Arbitrary computation in zero-knowledge
- **Client-Side WASM:** Browser-based proof generation
- **Hardware Acceleration:** GPU/FPGA for faster proving
- **Proof Delegation:** Third-party provers (privacy-preserving)
- **Cross-Chain Verification:** Verify proofs on blockchain
- **Verifiable Delay Functions:** Time-locked proofs

---

## 11. References

- Bulletproofs Paper: https://eprint.iacr.org/2017/1066.pdf
- dalek-cryptography/bulletproofs: https://github.com/dalek-cryptography/bulletproofs
- Groth16 Paper: https://eprint.iacr.org/2016/260.pdf
- circom (circuit compiler): https://docs.circom.io/
- ZKProof Standards: https://zkproof.org/
- Iden3 (identity + ZK): https://docs.iden3.io/
- PLONK Paper: https://eprint.iacr.org/2019/953.pdf
- Halo2: https://zcash.github.io/halo2/
- BBS+ Signatures: https://identity.foundation/bbs-signature/draft-irtf-cfrg-bbs-signatures.html
