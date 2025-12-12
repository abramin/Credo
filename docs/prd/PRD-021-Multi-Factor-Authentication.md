# PRD-021: Multi-Factor Authentication (MFA)

**Status:** Not Started
**Priority:** P1 (High)
**Owner:** Engineering Team
**Dependencies:** PRD-001 (Authentication), PRD-016 (Token Lifecycle), PRD-018 (Notifications)
**Last Updated:** 2025-12-12

---

## 1. Overview

### Problem Statement

Single-factor authentication (password/email only) is insufficient for regulated industries. Need MFA for:
- Healthcare (HIPAA compliance)
- Fintech (PCI-DSS, SOC 2)
- Government (NIST 800-63B requirements)

### Goals

- TOTP (Time-based One-Time Password, RFC 6238)
- SMS OTP delivery
- Email OTP delivery
- Biometric as 2nd factor (integrate PRD-013)
- Backup codes generation
- MFA enrollment & recovery flows
- Step-up authentication (re-auth for sensitive ops)
- Remember device functionality

### Non-Goals

- Hardware security keys (FIDO2/WebAuthn) - future
- Passkeys - future
- Push-based MFA (mobile app notifications)

---

## 2. Functional Requirements

### FR-1: MFA Enrollment

**Endpoint:** `POST /mfa/enroll`

**Input:**
```json
{
  "method": "totp" // or "sms", "email"
}
```

**Output (TOTP):**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,...",
  "backup_codes": ["12345678", "87654321", ...]
}
```

**Output (SMS/Email):**
```json
{
  "enrolled": true,
  "method": "sms",
  "masked_phone": "+1 (415) 555-****",
  "backup_codes": ["12345678", "87654321", ...]
}
```

---

### FR-2: MFA Challenge

**Endpoint:** `POST /mfa/challenge`

**Input:**
```json
{
  "session_id": "sess_abc123",
  "code": "123456"
}
```

**Output (Success):**
```json
{
  "verified": true,
  "access_token": "eyJhbGc...",
  "refresh_token": "ref_..."
}
```

**Error (Invalid Code):**
```json
{
  "error": "invalid_mfa_code",
  "attempts_remaining": 2
}
```

**Verification Controls (OWASP MFA & Authentication Cheat Sheets):**
- Enforce rate limits and lockouts: max 5 failed attempts per method per hour; lock method for 15 minutes after threshold and emit audit `mfa.locked_out`.
- Reject codes older than 90 seconds (TOTP 30s step, 1 step of clock skew) and block replay of successful codes within the same validity window.
- Use generic error responses to avoid method or account enumeration (do not reveal whether phone/email exists).
- Bind challenge to session and device fingerprint; require a new challenge if IP/UA changes between issuance and verification.
- Apply exponential backoff (250ms, 500ms, 1s…) on consecutive failures to slow credential stuffing.

---

### FR-3: Backup Codes

**Generate on enrollment:**
- 10 single-use backup codes
- 8 digits each
- Stored hashed (bcrypt)
- Shown once, user must save

**Usage:** Can be used instead of TOTP/SMS

---

### FR-4: Remember Device

**Cookie:** `mfa_device_token` (30-day TTL)

**Logic:**
- After successful MFA, set device token
- On subsequent logins from same device, skip MFA for 30 days
- Revoke on explicit logout or password change
- Device token must be signed, bound to user agent + key rotation, and validated for replay; do not store shared secrets in the cookie.

---

### FR-5: Step-Up Authentication

**Use Case:** Require re-authentication for sensitive operations

**Example:** Before deleting account, require recent MFA (< 5 min ago)

**Endpoint:** `POST /mfa/step-up`

**Middleware:**
```go
func RequireRecentMFA(maxAge time.Duration) middleware {
    // Check last_mfa_at from session
    // If > maxAge, require MFA challenge
}
```

---

## 3. Technical Requirements

### TR-1: MFA Models

```go
type MFAMethod struct {
    ID         string
    UserID     string
    Method     string // "totp", "sms", "email"
    Secret     string // Encrypted TOTP secret
    PhoneNumber string // For SMS
    Email      string // For Email OTP
    Verified   bool
    CreatedAt  time.Time
}

type BackupCode struct {
    ID         string
    UserID     string
    CodeHash   string // bcrypt hash
    Used       bool
    UsedAt     *time.Time
}
```

### TR-2: TOTP Implementation

```go
import "github.com/pquerna/otp/totp"

// Generate secret
secret, err := totp.Generate(totp.GenerateOpts{
    Issuer:      "Credo",
    AccountName: user.Email,
})

// Validate code
valid := totp.Validate(code, secret.Secret())
```

### TR-3: OWASP MFA Alignment
- **Secret handling:** TOTP seeds encrypted at rest with KMS and only decrypted in-memory for validation; backup codes stored as salted bcrypt hashes.
- **Channel strength:** SMS/email OTPs must carry a 6–8 digit numeric code with 5-minute TTL, include issuer/domain in the message, and avoid embedding links to reduce phishing risk.
- **Clock drift:** Allow at most ±1 step skew; monitor for repeated drift and alert.
- **Auditability:** Record `mfa_enrolled`, `mfa_verified`, `mfa_lockout`, and recovery events with method, IP, and device fingerprint.
- **User self-service hardening:** Enrollment requires primary-authenticated session + re-prompt of existing factor; recovery enforces backup code + out-of-band verification.

---

## 4. Implementation Steps

### Phase 1: TOTP (5-6 hours)
1. Implement TOTP enrollment
2. Generate QR codes
3. Implement verification
4. Add backup codes
5. Test with Google Authenticator

### Phase 2: SMS/Email OTP (3-4 hours)
1. Integrate with PRD-018 (Notifications)
2. Generate random 6-digit codes
3. Store codes with TTL (5 min)
4. Implement verification

### Phase 3: Remember Device (2-3 hours)
1. Generate device tokens
2. Set secure cookies
3. Validate on login
4. Test across browsers

---

## 5. Acceptance Criteria

- [ ] Users can enroll TOTP with QR code
- [ ] TOTP codes validated successfully
- [ ] SMS OTP delivered via Twilio
- [ ] Email OTP delivered via SendGrid
- [ ] Backup codes work as fallback
- [ ] Remember device skips MFA for 30 days
- [ ] Step-up authentication works for sensitive ops
- [ ] MFA required for admin accounts

---

## 6. API Examples

### Enroll TOTP
```bash
curl -X POST http://localhost:8080/mfa/enroll \
  -H "Authorization: Bearer eyJhbGc..." \
  -d '{"method":"totp"}'
```

### Verify MFA Challenge
```bash
curl -X POST http://localhost:8080/mfa/challenge \
  -d '{"session_id":"sess_abc","code":"123456"}'
```

---

## Revision History

| Version | Date       | Author       | Changes     |
| ------- | ---------- | ------------ | ----------- |
| 1.1     | 2025-12-12 | Product Team | Added OWASP-aligned MFA verification, lockout, and device token safeguards |
| 1.0     | 2025-12-12 | Product Team | Initial PRD |
