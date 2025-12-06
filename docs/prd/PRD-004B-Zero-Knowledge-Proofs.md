# PRD-004B: Zero-Knowledge Age Verification

**Status:** Not Started
**Priority:** P2 (Medium-High - Technically Impressive)
**Owner:** Engineering Team
**Dependencies:** PRD-004 (Verifiable Credentials) complete
**Last Updated:** 2025-12-06

---

## 1. Overview

### Problem Statement
Current age verification (PRD-004) requires revealing date of birth to verify age. Zero-knowledge proofs (ZKPs) allow proving "I am over 18" without revealing exact birthdate, providing stronger privacy guarantees.

### Goals
- Implement zero-knowledge proof of age without revealing DOB
- Use zk-SNARKs to prove `age >= 18` statement
- Generate and verify ZK proofs
- Demonstrate understanding of advanced cryptography
- Provide privacy-preserving alternative to standard VCs

### Non-Goals
- Production-grade ZK system (library complexity)
- Multi-statement proofs (just age for MVP)
- Recursive proofs or proof composition
- Hardware acceleration
- Formal verification of circuits

---

## 2. Background: Zero-Knowledge Proofs

**What are ZKPs?**
- Prove a statement is true without revealing why
- Example: "I know password X" without revealing X
- Example: "My age >= 18" without revealing exact age

**zk-SNARKs:**
- **Zero-Knowledge Succinct Non-Interactive Arguments of Knowledge**
- Small proof size (~200 bytes)
- Fast verification (~ms)
- Requires "trusted setup" (circuit-specific)

**Use Case: Age Verification**
```
Prover has: date_of_birth = 1990-05-15
Prover wants to prove: (current_date - date_of_birth) >= 18 years
Without revealing: exact date_of_birth

Proof: "I computed age >= 18" (cryptographically)
Verifier: Checks proof is valid (without learning age)
```

---

## 3. User Stories

**As a** privacy-conscious user
**I want to** prove I'm over 18 without revealing my birthdate
**So that** my privacy is protected

**As a** verifier
**I want to** trust the age claim without seeing DOB
**So that** I minimize PII exposure

**As a** developer
**I want to** understand ZKP fundamentals
**So that** I can apply them to other privacy problems

---

## 4. Functional Requirements

### FR-1: Generate ZK Proof of Age
**Endpoint:** `POST /zkp/prove-age`

**Description:** Generate zero-knowledge proof that user is over 18.

**Input:**
```json
{
  "date_of_birth": "1990-05-15",
  "threshold_age": 18
}
```

**Output (Success - 200):**
```json
{
  "proof": "base64-encoded-zk-proof",
  "public_inputs": {
    "current_date_hash": "hash-of-today",
    "threshold": 18
  },
  "proof_system": "groth16",
  "circuit_id": "age_verification_v1"
}
```

**Business Logic:**
1. Parse date_of_birth
2. Calculate age in days: `today - dob`
3. Convert to constraint system (circuit)
4. Generate witness (private inputs)
5. Compute zk-SNARK proof
6. Return proof + public inputs

---

### FR-2: Verify ZK Proof
**Endpoint:** `POST /zkp/verify-age`

**Description:** Verify zero-knowledge age proof.

**Input:**
```json
{
  "proof": "base64-encoded-zk-proof",
  "public_inputs": {
    "current_date_hash": "hash-of-today",
    "threshold": 18
  }
}
```

**Output (Success - 200):**
```json
{
  "valid": true,
  "verified_at": "2025-12-06T10:00:00Z",
  "statement": "Prover is >= 18 years old"
}
```

**Business Logic:**
1. Deserialize proof
2. Load verification key for circuit
3. Verify proof against public inputs
4. Return validity

---

### FR-3: Issue ZK-Backed Credential
**Enhancement to:** PRD-004

**Description:** Issue VC that includes ZK proof instead of plain claim.

**VC Structure:**
```json
{
  "credential_id": "vc_zk_abc123",
  "type": "AgeOver18",
  "subject": "user_123",
  "issuer": "id-gateway",
  "issued_at": "2025-12-06T10:00:00Z",
  "claims": {
    "age_proof": {
      "proof": "base64-zk-proof",
      "public_inputs": {...},
      "circuit_id": "age_verification_v1"
    }
  },
  "privacy_enhanced": true
}
```

**Verification:** Verify ZK proof instead of trusting issuer claim.

---

## 5. Technical Requirements

### TR-1: ZK Circuit Definition

**Concept:** Circuit expresses computation as arithmetic constraints.

**Age Verification Circuit (Pseudo-code):**
```
Inputs:
  - Private: date_of_birth (as integer days since epoch)
  - Public: current_date_hash, threshold_age

Computation:
  age_in_days = current_date - date_of_birth
  age_in_years = age_in_days / 365
  
Constraint:
  age_in_years >= threshold_age
  
Output:
  If constraint satisfied → proof generated
  Else → proof fails
```

### TR-2: Go Library Integration

**Library:** `github.com/consensys/gnark` (Go zk-SNARK library)

**Circuit Definition:**
```go
import (
    "github.com/consensys/gnark/frontend"
    "github.com/consensys/gnark/std/math/cmp"
)

type AgeCircuit struct {
    DateOfBirth     frontend.Variable `gnark:",secret"`
    CurrentDate     frontend.Variable `gnark:",public"`
    ThresholdAge    frontend.Variable `gnark:",public"`
    AgeInDays       frontend.Variable `gnark:",secret"`
}

func (circuit *AgeCircuit) Define(api frontend.API) error {
    // Compute age in days
    circuit.AgeInDays = api.Sub(circuit.CurrentDate, circuit.DateOfBirth)
    
    // Convert to years (simplified: 365 days/year)
    ageInYears := api.Div(circuit.AgeInDays, 365)
    
    // Assert age >= threshold
    api.AssertIsLessOrEqual(circuit.ThresholdAge, ageInYears)
    
    return nil
}
```

### TR-3: Proof Generation

**Location:** `internal/zkp/prover.go` (new package)

```go
import (
    "github.com/consensys/gnark-crypto/ecc"
    "github.com/consensys/gnark/backend/groth16"
    "github.com/consensys/gnark/frontend"
    "github.com/consensys/gnark/frontend/cs/r1cs"
)

type Prover struct {
    provingKey groth16.ProvingKey
    circuit    frontend.Circuit
}

func (p *Prover) ProveAge(dob time.Time, threshold int) ([]byte, error) {
    // Prepare witness (private + public inputs)
    dobDays := int(dob.Unix() / 86400)
    currentDays := int(time.Now().Unix() / 86400)
    
    witness := AgeCircuit{
        DateOfBirth:  dobDays,
        CurrentDate:  currentDays,
        ThresholdAge: threshold,
    }
    
    // Generate proof
    proof, err := groth16.Prove(p.circuit, p.provingKey, &witness)
    if err != nil {
        return nil, err
    }
    
    // Serialize proof
    return proof.MarshalBinary()
}
```

### TR-4: Proof Verification

**Location:** `internal/zkp/verifier.go` (new file)

```go
type Verifier struct {
    verifyingKey groth16.VerifyingKey
}

func (v *Verifier) VerifyAge(proofBytes []byte, publicInputs map[string]int) (bool, error) {
    // Deserialize proof
    proof := groth16.NewProof(ecc.BN254)
    err := proof.UnmarshalBinary(proofBytes)
    if err != nil {
        return false, err
    }
    
    // Prepare public witness
    publicWitness := AgeCircuit{
        CurrentDate:  publicInputs["current_date"],
        ThresholdAge: publicInputs["threshold"],
    }
    
    // Verify proof
    err = groth16.Verify(proof, v.verifyingKey, &publicWitness)
    return err == nil, err
}
```

### TR-5: Trusted Setup (One-Time)

**Script:** `scripts/zkp_setup.go`

```go
func runTrustedSetup() error {
    // Define circuit
    var circuit AgeCircuit
    
    // Compile circuit to R1CS
    r1cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
    if err != nil {
        return err
    }
    
    // Run setup (generates proving + verifying keys)
    pk, vk, err := groth16.Setup(r1cs)
    if err != nil {
        return err
    }
    
    // Save keys to disk
    saveProvingKey(pk, "zkp/proving_key.bin")
    saveVerifyingKey(vk, "zkp/verifying_key.bin")
    
    return nil
}
```

**Note:** Setup is circuit-specific and only needs to run once.

---

## 6. Implementation Steps

### Phase 1: Circuit Design & Setup (4-6 hours)
1. Install gnark library
2. Define age verification circuit
3. Run trusted setup
4. Test circuit with sample inputs

### Phase 2: Proof Generation (3-4 hours)
1. Implement Prover service
2. Generate proofs for valid/invalid ages
3. Serialize proofs

### Phase 3: Proof Verification (2-3 hours)
1. Implement Verifier service
2. Verify valid proofs pass
3. Verify invalid proofs fail

### Phase 4: HTTP Endpoints (2 hours)
1. Implement /zkp/prove-age
2. Implement /zkp/verify-age
3. Return structured responses

### Phase 5: VC Integration (2-3 hours)
1. Add ZK proof to VC claims
2. Store ZK-backed credentials
3. Verify ZK-backed credentials

### Phase 6: Testing & Documentation (3-4 hours)
1. Unit tests for circuit correctness
2. Integration tests for full flow
3. Document ZKP concepts clearly
4. Performance benchmarks

---

## 7. Acceptance Criteria

- [ ] Age verification circuit compiles successfully
- [ ] Trusted setup generates keys
- [ ] Valid age proofs verify correctly
- [ ] Invalid age proofs fail verification
- [ ] Under-18 users cannot generate valid proofs
- [ ] Proof size < 500 bytes
- [ ] Verification time < 10ms
- [ ] Proofs can be embedded in VCs
- [ ] Documentation explains ZKP clearly for non-experts
- [ ] No date-of-birth leaks in proof or public inputs

---

## 8. Testing

```bash
# Run trusted setup (one-time)
go run scripts/zkp_setup.go

# Generate proof (user is 25 years old)
curl -X POST http://localhost:8080/zkp/prove-age \
  -d '{"date_of_birth": "1999-01-01", "threshold_age": 18}'
# Expected: {"proof": "base64...", "public_inputs": {...}}

# Verify proof
curl -X POST http://localhost:8080/zkp/verify-age \
  -d '{"proof": "base64...", "public_inputs": {...}}'
# Expected: {"valid": true}

# Try to prove age for 17-year-old (should fail to generate proof)
curl -X POST http://localhost:8080/zkp/prove-age \
  -d '{"date_of_birth": "2008-01-01", "threshold_age": 18}'
# Expected: 400 Bad Request (circuit constraints not satisfied)
```

---

## 9. Performance Considerations

**Proof Generation:**
- Time: 1-5 seconds (circuit complexity dependent)
- Memory: ~500MB (circuit compilation)

**Proof Verification:**
- Time: 5-10ms (very fast)
- Memory: ~10MB (verification key)

**Proof Size:**
- Groth16: ~200 bytes (constant size)

**Optimization:**
- Pre-compile circuit
- Cache proving/verifying keys
- Batch proofs (future)

---

## 10. Security Considerations

**Trusted Setup:**
- Setup generates "toxic waste" (must be destroyed)
- If toxic waste leaks, fake proofs possible
- Mitigation: Multi-party computation (MPC) setup

**Side-Channel Attacks:**
- Timing attacks on proof generation
- Mitigation: Constant-time operations

**Circuit Bugs:**
- Incorrect circuit → fake proofs accepted
- Mitigation: Formal verification, peer review, tests

---

## 11. Future Enhancements

- Multi-statement proofs (age + citizenship + not sanctioned)
- Recursive proofs (prove proof is valid)
- Range proofs (age in 18-65 range)
- Universal trusted setup (use existing ceremonies)
- Hardware acceleration (GPU proof generation)
- Privacy-preserving credential presentation

---

## 12. Resume Talking Points

**What This Demonstrates:**
- Advanced cryptography (ZKPs, zk-SNARKs)
- Privacy engineering
- Cutting-edge technology (Groth16)
- Ability to learn hard concepts

**Interview Answers:**
- "How do ZKPs work?" → Prover converts statement to arithmetic circuit, generates proof, verifier checks without learning secrets
- "What's trusted setup?" → One-time ceremony to generate proving/verifying keys, must destroy intermediate values
- "Limitations?" → Requires circuit per statement, trusted setup has trust assumptions, proof generation slow

---

## 13. References

- [Groth16 Paper](https://eprint.iacr.org/2016/260.pdf)
- [gnark Documentation](https://docs.gnark.consensys.net/)
- [ZKP Primitives](https://github.com/matter-labs/awesome-zero-knowledge-proofs)
- [Zcash Ceremony](https://z.cash/technology/paramgen/) - Real-world trusted setup

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-06 | Engineering Team | Initial PRD |
