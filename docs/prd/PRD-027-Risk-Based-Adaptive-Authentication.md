# PRD-027: Risk-Based Adaptive Authentication

**Status:** Not Started
**Priority:** P1 (High)
**Owner:** Engineering Team
**Dependencies:** PRD-001 (Authentication), PRD-023 (Fraud Detection), PRD-005B (Cerbos), PRD-021 (MFA)
**Last Updated:** 2025-12-12

---

## 1. Overview

### Problem Statement

Current authentication is binary (authenticated or not). Risk scoring (PRD-023) generates continuous risk assessments, but there's no automated system to translate risk scores into adaptive actions. Need a policy-driven layer that:

- Takes risk score + event type as input
- Decides appropriate action (allow, require MFA, deny, soft-lock)
- Enforces action through existing auth/policy systems
- Allows easy tuning without code changes

This adds **real value to an identity gateway** by making authentication context-aware.

### Goals

- **Risk to action matrix:** Configure risk thresholds and actions per event type
- **Adaptive authentication:** Automatically escalate authentication requirements based on risk
- **Step-up authentication:** Require MFA/re-auth for high-risk operations
- **Soft session locking:** Temporarily restrict session without full logout
- **Config-driven:** No code changes for policy updates (DB table or YAML)
- **Cerbos integration:** Use PRD-005B policy engine for decision enforcement
- **Shadow mode:** Test policies without enforcement (observability first)

### Non-Goals

- Machine learning-based risk models (see PRD-007B)
- User behavior analytics beyond risk scores
- Geofencing (location-based hard blocks)
- Biometric step-up (covered in PRD-021)
- Real-time risk recalculation (use latest score from PRD-023)

---

## 2. User Stories

**As a security engineer**
**I want to** configure risk thresholds that trigger MFA requirements
**So that** high-risk logins are automatically challenged without manual intervention

**As a system operator**
**I want to** soft-lock suspicious sessions without fully logging users out
**So that** I can investigate while minimizing user disruption

**As a compliance officer**
**I want to** enforce step-up authentication for sensitive operations
**So that** we meet regulatory requirements for high-risk transactions

**As a developer**
**I want to** test new risk policies in shadow mode
**So that** I can validate rules before enforcing them in production

---

## 3. Functional Requirements

### FR-1: Risk to Action Matrix

**Description:** Mapping table from (risk score band, event type) → action

**Matrix Example:**

| Event Type          | Risk 0-20 (Low) | Risk 21-50 (Med) | Risk 51-75 (High) | Risk 76-100 (Critical) |
| ------------------- | --------------- | ---------------- | ----------------- | ---------------------- |
| **Login**           | Allow           | Allow + Log      | Require MFA       | Deny + Soft-Lock       |
| **Consent Grant**   | Allow           | Allow            | Require Re-Auth   | Deny + Review          |
| **VC Issuance**     | Allow           | Require MFA      | Require MFA       | Deny + Alert           |
| **Data Export**     | Allow           | Require Re-Auth  | Require MFA       | Deny + Manual Review   |
| **Password Change** | Allow           | Require Re-Auth  | Require MFA       | Deny + Support Ticket  |
| **Session Create**  | Allow           | Allow + Monitor  | Challenge         | Deny                   |

**Actions:**

- **Allow:** Permit operation (no additional checks)
- **Allow + Log:** Permit but emit warning log
- **Allow + Monitor:** Permit and add to review queue
- **Require MFA:** Demand MFA challenge before proceeding (PRD-021)
- **Require Re-Auth:** Demand password re-entry (session < 5 min old)
- **Deny:** Block operation with 403 Forbidden
- **Deny + Soft-Lock:** Block and lock session for 15 minutes
- **Deny + Alert:** Block and notify security team
- **Deny + Manual Review:** Block and add to admin review queue

**Configuration:**

Stored in PostgreSQL table or YAML file:

```sql
CREATE TABLE risk_action_policies (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,              -- "login", "consent_grant", "vc_issuance"
    risk_min INT NOT NULL,                 -- Lower bound (0-100)
    risk_max INT NOT NULL,                 -- Upper bound (0-100)
    action TEXT NOT NULL,                  -- "allow", "require_mfa", "deny"
    metadata JSONB,                        -- Additional params (lockout_duration, etc.)
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_policy UNIQUE (event_type, risk_min, risk_max)
);

CREATE INDEX idx_risk_policies_event ON risk_action_policies(event_type);
```

**Example Rows:**

```sql
INSERT INTO risk_action_policies (event_type, risk_min, risk_max, action, metadata) VALUES
  ('login', 0, 20, 'allow', '{}'),
  ('login', 21, 50, 'allow', '{"log_level": "warn"}'),
  ('login', 51, 75, 'require_mfa', '{}'),
  ('login', 76, 100, 'deny', '{"soft_lock": true, "duration_min": 15}'),

  ('vc_issuance', 0, 20, 'allow', '{}'),
  ('vc_issuance', 21, 50, 'require_mfa', '{}'),
  ('vc_issuance', 51, 100, 'deny', '{"alert": true}');
```

**Evaluation Logic:**

```go
func (s *AdaptiveAuthService) EvaluateAction(ctx context.Context, eventType string, riskScore int) (*Action, error) {
    // Query policy table
    policies, err := s.store.FindPolicies(ctx, eventType)
    if err != nil {
        return nil, err
    }

    // Find matching policy (risk_min <= score <= risk_max)
    for _, policy := range policies {
        if policy.RiskMin <= riskScore && riskScore <= policy.RiskMax {
            return &Action{
                Type:     policy.Action,
                Metadata: policy.Metadata,
                PolicyID: policy.ID,
            }, nil
        }
    }

    // Default: allow (fail open for safety)
    return &Action{Type: "allow"}, nil
}
```

---

### FR-2: Adaptive Authentication Actions

**Scope:** All authenticated endpoints

**Description:** Enforce actions determined by risk-to-action matrix.

#### FR-2.1: Allow (Baseline)

**Action:** Continue processing normally

**Implementation:** No-op, log for audit

#### FR-2.2: Allow + Log/Monitor

**Action:** Continue but emit warning log or add to review queue

**Implementation:**

- Emit audit event: `high_risk_operation_allowed`
- Add to admin review queue (PostgreSQL table)
- Notify security team if configured (Slack/PagerDuty)

#### FR-2.3: Require MFA

**Action:** Demand MFA challenge before proceeding

**Flow:**

1. Check if user has enrolled MFA method (PRD-021)
2. If yes:
   - Return 401 with `X-Challenge: mfa`
   - Include challenge ID in response
   - Client re-submits request with MFA code
3. If no:
   - Fallback to "Require Re-Auth" (password re-entry)

**Response (401):**

```json
{
  "error": "mfa_required",
  "message": "This operation requires multi-factor authentication.",
  "challenge_id": "chall_abc123",
  "challenge_type": "totp", // or "sms", "email"
  "session_id": "sess_xyz789"
}
```

**Client Flow:**

```bash
# 1. Initial request (high risk detected)
POST /vc/issue → 401 mfa_required

# 2. Client prompts user for MFA code

# 3. Client re-submits with MFA
POST /mfa/challenge
{
  "challenge_id": "chall_abc123",
  "code": "123456"
}
→ 200 OK {verified: true, token: "stepup_token_xyz"}

# 4. Client retries original request with step-up token
POST /vc/issue
Authorization: Bearer stepup_token_xyz
→ 200 OK (VC issued)
```

**Step-Up Token:**

- Short-lived (5 minutes)
- Single-use
- Bound to original session + operation
- Stored in Redis: `stepup:{session_id}:{operation}` → token (TTL: 5min)

#### FR-2.4: Require Re-Auth

**Action:** Demand password re-entry (lighter than MFA)

**Use Case:** Medium-risk operations, user has no MFA enrolled

**Flow:**

1. Return 401 with `X-Challenge: password`
2. Client prompts for password
3. Client calls `/auth/re-auth` with password
4. Server validates password, issues step-up token
5. Client retries original request with step-up token

**Response (401):**

```json
{
  "error": "re_auth_required",
  "message": "Please re-enter your password to confirm this action.",
  "challenge_id": "chall_xyz456",
  "session_id": "sess_xyz789"
}
```

#### FR-2.5: Deny

**Action:** Block operation with 403 Forbidden

**Response:**

```json
{
  "error": "operation_denied",
  "message": "This operation cannot be performed due to security concerns.",
  "reason": "high_risk_score",
  "risk_score": 85,
  "support_contact": "security@example.com"
}
```

**Audit:**

- Emit event: `operation_denied_risk`
- Include risk score, factors, policy ID

#### FR-2.6: Deny + Soft-Lock

**Action:** Block operation and lock session temporarily

**Description:** Session is restricted for N minutes (default: 15). User stays authenticated but all write operations return 403.

**Implementation:**

```sql
CREATE TABLE session_locks (
    session_id UUID PRIMARY KEY,
    reason TEXT,
    locked_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    locked_by TEXT DEFAULT 'system'
);

CREATE INDEX idx_session_locks_expires ON session_locks(expires_at);
```

**Middleware Check:**

```go
func SoftLockMiddleware(store SessionLockStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            sessionID := getSessionID(r.Context())

            lock, err := store.GetLock(r.Context(), sessionID)
            if err == nil && lock.ExpiresAt.After(time.Now()) {
                // Session is locked
                http.Error(w, "Session temporarily locked due to suspicious activity", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Unlock:**

- Automatic: Lock expires after duration (15 min default)
- Manual: Admin can unlock via `/admin/sessions/{id}/unlock`

**Response (403):**

```json
{
  "error": "session_locked",
  "message": "Your session has been temporarily locked due to suspicious activity.",
  "locked_until": "2025-12-12T11:15:00Z",
  "reason": "high_risk_score",
  "unlock_instructions": "Contact support or wait until lock expires."
}
```

---

### FR-3: Step-Up Authentication Triggers

**Description:** Define when step-up auth is required independent of risk score

**Triggers:**

| Trigger                            | Step-Up Requirement                      |
| ---------------------------------- | ---------------------------------------- |
| **Password Change**                | Always require MFA or re-auth            |
| **Email Change**                   | Always require MFA                       |
| **MFA Enrollment**                 | Require password re-entry                |
| **Delete Account**                 | Require MFA + admin approval             |
| **Export All Data (GDPR)**         | Require MFA (< 5 min old)                |
| **Revoke All Sessions**            | Require MFA                              |
| **Change Payment Method**          | Require MFA (if fintech mode)            |
| **Issue Verifiable Credential**    | Check risk score (matrix)                |
| **Registry Lookup (>10/hour)**     | Require recent MFA (< 10 min)            |

**Implementation:**

```go
// Per-handler enforcement
func handlePasswordChange(w http.ResponseWriter, r *http.Request) {
    // Always require step-up
    if !hasRecentStepUp(r.Context(), 5*time.Minute) {
        requireStepUp(w, "mfa")
        return
    }

    // Proceed with password change
    // ...
}
```

**Session Metadata:**

Store last step-up time in session:

```go
type Session struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    LastStepUpAt  *time.Time  // When user last completed MFA/re-auth
    // ... other fields
}
```

---

### FR-4: Cerbos Policy Integration

**Description:** Use PRD-005B Cerbos policy engine to enforce risk-based decisions

**Cerbos Policy Example:**

```yaml
# policies/adaptive_auth.yaml
apiVersion: api.cerbos.dev/v1
resourcePolicy:
  version: "1.0"
  resource: "auth_session"
  rules:
    - actions: ["vc_issue"]
      effect: EFFECT_ALLOW
      condition:
        match:
          all:
            of:
              - expr: "request.aux_data.risk_score < 21"

    - actions: ["vc_issue"]
      effect: EFFECT_DENY
      condition:
        match:
          all:
            of:
              - expr: "request.aux_data.risk_score > 75"
      output:
        expr: '{"action": "deny", "soft_lock": true}'

    - actions: ["vc_issue"]
      effect: EFFECT_ALLOW
      condition:
        match:
          all:
            of:
              - expr: "request.aux_data.risk_score >= 21"
              - expr: "request.aux_data.risk_score <= 75"
              - expr: "request.aux_data.has_recent_mfa == true"
```

**Request Context:**

```go
func callCerbos(ctx context.Context, sessionID, action string, riskScore int) (*cerbos.Decision, error) {
    req := cerbos.CheckResourcesRequest{
        Principal: cerbos.Principal{
            ID:   sessionID,
            Roles: []string{"authenticated_user"},
        },
        Resource: cerbos.Resource{
            Kind: "auth_session",
            ID:   sessionID,
            Attr: map[string]interface{}{
                "risk_score":      riskScore,
                "has_recent_mfa":  hasRecentMFA(ctx, sessionID),
                "session_age_min": getSessionAge(ctx, sessionID),
            },
        },
        Actions: []string{action},
    }

    return cerbosClient.CheckResources(ctx, req)
}
```

**Advantages:**

- Policy versioning (git-based change control)
- Testable policies (Cerbos test framework)
- Separation of logic from code
- Auditable policy changes

---

### FR-5: Shadow Mode (Testing Without Enforcement)

**Description:** Test new policies without blocking users

**How It Works:**

1. Enable shadow mode: `shadow_mode: true` in policy metadata
2. Risk evaluator runs as normal
3. Action is determined but NOT enforced
4. Log what would have happened
5. Emit metrics for decision distribution

**Configuration:**

```sql
ALTER TABLE risk_action_policies ADD COLUMN shadow_mode BOOLEAN DEFAULT false;

-- Enable shadow mode for testing
UPDATE risk_action_policies
SET shadow_mode = true
WHERE event_type = 'vc_issuance' AND risk_min = 51;
```

**Logging:**

```json
{
  "timestamp": "2025-12-12T10:30:00Z",
  "level": "info",
  "event": "shadow_mode_decision",
  "session_id": "sess_abc123",
  "event_type": "vc_issuance",
  "risk_score": 65,
  "would_have_action": "require_mfa",
  "actual_action": "allow",
  "policy_id": "policy_xyz789"
}
```

**Metrics:**

```
# Counter: Shadow mode decisions by action type
adaptive_auth_shadow_decisions_total{action="allow|require_mfa|deny"}

# Comparison: What would have been denied vs. allowed
adaptive_auth_shadow_denial_rate
```

**Use Case:**

- Test risk thresholds before enforcement
- Validate new policies don't break UX
- A/B test different action strategies

---

### FR-6: Admin Override & Manual Review Queue

**Description:** Allow admins to override locked sessions and review denied operations

#### FR-6.1: Manual Review Queue

**Table:**

```sql
CREATE TABLE manual_review_queue (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    risk_score INT NOT NULL,
    denied_at TIMESTAMPTZ NOT NULL,
    reviewed_at TIMESTAMPTZ,
    reviewer_id UUID,
    decision TEXT,  -- "approve", "deny", "escalate"
    notes TEXT,
    status TEXT DEFAULT 'pending'  -- "pending", "approved", "denied"
);

CREATE INDEX idx_review_queue_status ON manual_review_queue(status);
```

**Workflow:**

1. High-risk operation denied with metadata `{manual_review: true}`
2. Add row to `manual_review_queue`
3. Notify security team (Slack, email)
4. Admin reviews via `/admin/review-queue`
5. Admin approves → issue step-up token, notify user
6. Admin denies → session remains locked, user notified

**API:**

```bash
# List pending reviews
GET /admin/review-queue?status=pending

# Approve operation
POST /admin/review-queue/{id}/approve
{
  "notes": "Verified user identity via phone call"
}

# Deny operation
POST /admin/review-queue/{id}/deny
{
  "notes": "Unable to verify, recommend password reset"
}
```

#### FR-6.2: Session Unlock

**Endpoint:** `POST /admin/sessions/{session_id}/unlock`

**Input:**

```json
{
  "reason": "False positive, user verified",
  "notify_user": true
}
```

**Action:**

- Remove row from `session_locks` table
- Emit audit event: `session_unlocked_by_admin`
- Optionally notify user via email/SMS

---

## 4. Technical Requirements

### TR-1: Data Models

**AdaptiveAuthService:**

```go
type AdaptiveAuthService struct {
    policyStore    PolicyStore
    sessionStore   SessionStore
    lockStore      SessionLockStore
    cerbosClient   *cerbos.Client
    auditor        audit.Publisher
}

type PolicyStore interface {
    FindPolicies(ctx context.Context, eventType string) ([]*RiskActionPolicy, error)
    CreatePolicy(ctx context.Context, policy *RiskActionPolicy) error
    UpdatePolicy(ctx context.Context, policy *RiskActionPolicy) error
    DeletePolicy(ctx context.Context, id string) error
}

type SessionLockStore interface {
    CreateLock(ctx context.Context, lock *SessionLock) error
    GetLock(ctx context.Context, sessionID string) (*SessionLock, error)
    DeleteLock(ctx context.Context, sessionID string) error
}

type RiskActionPolicy struct {
    ID         string
    EventType  string
    RiskMin    int
    RiskMax    int
    Action     string
    Metadata   map[string]interface{}
    ShadowMode bool
    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type SessionLock struct {
    SessionID string
    Reason    string
    LockedAt  time.Time
    ExpiresAt time.Time
    LockedBy  string  // "system" or admin user ID
}

type Action struct {
    Type         string                 // "allow", "require_mfa", "deny"
    Metadata     map[string]interface{} // Additional params
    PolicyID     string
    ShadowMode   bool
}
```

### TR-2: Middleware Chain

**Order of Execution:**

1. **Authentication** (extract session, user)
2. **Risk Scoring** (PRD-023, calculate risk score)
3. **Soft Lock Check** (check if session is locked)
4. **Adaptive Auth Evaluation** (evaluate risk-to-action matrix)
5. **Action Enforcement** (require MFA, deny, etc.)
6. **Handler Execution** (if allowed)

**Example:**

```go
router.Use(
    AuthenticationMiddleware,      // PRD-001
    RiskScoringMiddleware,          // PRD-023
    SoftLockMiddleware,             // PRD-027
    AdaptiveAuthMiddleware,         // PRD-027
)

func AdaptiveAuthMiddleware(service *AdaptiveAuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            riskScore := getRiskScore(r.Context())
            eventType := getEventType(r.Context()) // "vc_issuance", "login", etc.

            action, err := service.EvaluateAction(r.Context(), eventType, riskScore)
            if err != nil {
                logger.Error("adaptive auth eval failed", "error", err)
                next.ServeHTTP(w, r)
                return
            }

            // Shadow mode: log but don't enforce
            if action.ShadowMode {
                logger.Info("shadow mode decision", "action", action.Type)
                next.ServeHTTP(w, r)
                return
            }

            // Enforce action
            switch action.Type {
            case "allow":
                next.ServeHTTP(w, r)
            case "require_mfa":
                requireMFAChallenge(w, r, action)
            case "deny":
                denyOperation(w, r, action)
            default:
                next.ServeHTTP(w, r)
            }
        })
    }
}
```

### TR-3: Event Type Detection

**How to Determine Event Type:**

```go
func getEventType(r *http.Request) string {
    switch r.URL.Path {
    case "/auth/authorize":
        return "login"
    case "/auth/consent":
        return "consent_grant"
    case "/vc/issue":
        return "vc_issuance"
    case "/me/data-export":
        return "data_export"
    case "/me/password":
        return "password_change"
    default:
        return "generic_operation"
    }
}
```

**Alternative:** Use route metadata

```go
router.HandleFunc("/vc/issue", handleVCIssue).
    Metadata("event_type", "vc_issuance")
```

### TR-4: Step-Up Token Generation

**Format:**

- JWT with claims: `{session_id, operation, exp, iat}`
- Short-lived: 5 minutes
- Single-use: Invalidate after use
- Signature: HS256 with secret key

**Storage:**

```
Redis Key: stepup:{session_id}:{operation} → token (TTL: 5min)
```

**Validation:**

```go
func validateStepUpToken(ctx context.Context, sessionID, operation, token string) error {
    // Parse JWT
    claims, err := parseJWT(token)
    if err != nil {
        return errors.New("invalid token")
    }

    // Check expiry
    if claims.ExpiresAt.Before(time.Now()) {
        return errors.New("token expired")
    }

    // Check single-use
    key := fmt.Sprintf("stepup:%s:%s", sessionID, operation)
    exists, _ := redis.Exists(ctx, key)
    if !exists {
        return errors.New("token already used")
    }

    // Delete token (single-use)
    redis.Del(ctx, key)

    return nil
}
```

---

## 5. API Specifications

### Endpoint Summary

| Endpoint                            | Method | Auth     | Purpose                          |
| ----------------------------------- | ------ | -------- | -------------------------------- |
| `/mfa/challenge`                    | POST   | Session  | Submit MFA code for step-up      |
| `/auth/re-auth`                     | POST   | Session  | Re-enter password for step-up    |
| `/admin/sessions/{id}/unlock`       | POST   | Admin    | Unlock soft-locked session       |
| `/admin/review-queue`               | GET    | Admin    | List manual review items         |
| `/admin/review-queue/{id}/approve`  | POST   | Admin    | Approve denied operation         |
| `/admin/review-queue/{id}/deny`     | POST   | Admin    | Deny operation permanently       |
| `/admin/policies`                   | GET    | Admin    | List risk action policies        |
| `/admin/policies`                   | POST   | Admin    | Create new policy                |
| `/admin/policies/{id}`              | PUT    | Admin    | Update policy                    |
| `/admin/policies/{id}`              | DELETE | Admin    | Delete policy                    |

### Example: Require MFA Flow

**Step 1: User attempts high-risk operation**

```bash
POST /vc/issue
Authorization: Bearer at_abc123
{
  "credential_type": "ProofOfAddress",
  "subject": "did:example:123"
}

# Response: 401 Unauthorized
{
  "error": "mfa_required",
  "message": "This operation requires multi-factor authentication.",
  "challenge_id": "chall_xyz789",
  "challenge_type": "totp",
  "session_id": "sess_abc123"
}
```

**Step 2: User submits MFA code**

```bash
POST /mfa/challenge
{
  "challenge_id": "chall_xyz789",
  "code": "654321"
}

# Response: 200 OK
{
  "verified": true,
  "step_up_token": "eyJhbGc...",
  "expires_at": "2025-12-12T10:35:00Z"
}
```

**Step 3: User retries operation with step-up token**

```bash
POST /vc/issue
Authorization: Bearer at_abc123
X-Step-Up-Token: eyJhbGc...
{
  "credential_type": "ProofOfAddress",
  "subject": "did:example:123"
}

# Response: 200 OK
{
  "credential": {...},
  "issued_at": "2025-12-12T10:30:00Z"
}
```

---

## 6. Security Requirements

### SR-1: Step-Up Token Security

- Tokens must be short-lived (5 minutes max)
- Tokens must be single-use
- Tokens must be bound to session + operation
- Tokens must include cryptographic signature
- Tokens invalidated after use or on session termination

### SR-2: Soft Lock Security

- Locks must auto-expire (max 24 hours)
- Locked sessions cannot perform write operations
- Lock bypass requires admin authentication
- Lock actions audited (who, when, why)

### SR-3: Policy Security

- Policy changes require admin role
- Policy changes audited with diff
- Shadow mode cannot be disabled without review
- Default action is "allow" (fail open for safety, configurable)

---

## 7. Observability Requirements

### Logging

**Events to Log:**

- `adaptive_auth_action_taken` (info) - Action enforced
- `mfa_required` (info) - MFA challenge issued
- `session_soft_locked` (warning) - Session locked
- `session_unlocked_by_admin` (info) - Admin override
- `shadow_mode_decision` (info) - Shadow mode test
- `policy_updated` (audit) - Policy config changed

**Log Format:**

```json
{
  "timestamp": "2025-12-12T10:30:00Z",
  "level": "warning",
  "event": "session_soft_locked",
  "session_id": "sess_abc123",
  "user_id": "user_xyz789",
  "risk_score": 82,
  "event_type": "vc_issuance",
  "action": "deny",
  "policy_id": "policy_123",
  "lock_duration_min": 15
}
```

### Metrics

```
# Counter: Actions taken by type
adaptive_auth_actions_total{action="allow|require_mfa|deny|soft_lock"}

# Counter: MFA challenges issued
adaptive_auth_mfa_challenges_total

# Counter: Sessions locked
adaptive_auth_sessions_locked_total

# Gauge: Currently locked sessions
adaptive_auth_locked_sessions

# Histogram: Risk score distribution per action
adaptive_auth_risk_score_per_action{action="..."}

# Counter: Shadow mode decisions
adaptive_auth_shadow_decisions_total{action="..."}

# Counter: Manual reviews
adaptive_auth_manual_reviews_total{decision="approve|deny|escalate"}
```

---

## 8. Testing Requirements

### Unit Tests

- [ ] Test policy evaluation (risk score → action mapping)
- [ ] Test step-up token generation and validation
- [ ] Test soft lock creation and expiry
- [ ] Test shadow mode (logs but doesn't enforce)
- [ ] Test default action (fail open)

### Integration Tests

- [ ] End-to-end MFA step-up flow
- [ ] Session soft-lock enforcement
- [ ] Admin unlock flow
- [ ] Manual review queue workflow
- [ ] Cerbos policy integration
- [ ] Shadow mode with real traffic

### Load Tests

- [ ] Policy evaluation latency <10ms
- [ ] Step-up token validation <5ms
- [ ] Soft lock check <5ms

---

## 9. Implementation Steps

### Phase 1: Foundation (Week 1)

1. Create PostgreSQL tables (policies, locks, review queue)
2. Implement PolicyStore and SessionLockStore
3. Build AdaptiveAuthService
4. Implement policy evaluation logic
5. Write unit tests

### Phase 2: Action Enforcement (Week 2)

1. Implement "Require MFA" action
2. Implement "Require Re-Auth" action
3. Implement "Deny" action
4. Implement "Soft Lock" action
5. Build step-up token generation/validation

### Phase 3: Middleware Integration (Week 3)

1. Build AdaptiveAuthMiddleware
2. Integrate with RiskScoringMiddleware (PRD-023)
3. Add event type detection
4. Test with real requests
5. Integration tests

### Phase 4: Admin Tools (Week 4)

1. Build manual review queue APIs
2. Build session unlock API
3. Build policy management APIs
4. Extend PRD-026 admin UI
5. Test admin workflows

### Phase 5: Cerbos Integration (Week 5)

1. Write Cerbos policies for adaptive auth
2. Integrate Cerbos client
3. Test policy evaluation via Cerbos
4. Shadow mode testing
5. Production rollout

---

## 10. Acceptance Criteria

- [ ] Risk-to-action matrix configured and operational
- [ ] MFA step-up flow works for high-risk operations
- [ ] Soft session locking blocks operations for duration
- [ ] Shadow mode logs decisions without enforcement
- [ ] Admin can unlock sessions manually
- [ ] Manual review queue functional
- [ ] Cerbos policies integrated (optional for MVP)
- [ ] Metrics and dashboards operational
- [ ] Latency <10ms for policy evaluation
- [ ] Code passes security review

---

## 11. Dependencies & Blockers

### Dependencies

- PRD-001: Authentication (session management)
- PRD-023: Fraud Detection (risk scoring)
- PRD-021: MFA (step-up authentication methods)
- PRD-005B: Cerbos (optional policy enforcement)

**External Libraries:**

- PostgreSQL (already in stack)
- Redis (already in stack)
- Cerbos (optional, can use DB policies)

### Potential Blockers

- Cerbos learning curve (mitigate: start with DB policies)
- UX complexity (MFA challenge flow)
- False positives (shadow mode testing critical)

---

## 12. Future Enhancements (Out of Scope)

- ML-based risk thresholds (auto-tuning)
- User-specific risk tolerance profiles
- Geofencing (location-based hard blocks)
- Behavioral challenge questions
- Anomaly-based step-up (not just risk score)
- Integration with external threat intel feeds

---

## 13. Open Questions

1. **Default Action:** Fail open (allow) or fail closed (deny)?
   - **Recommendation:** Fail open for safety, but log for review

2. **Soft Lock Duration:** 15 minutes sufficient?
   - **Recommendation:** Configurable per policy (5-60 min)

3. **Manual Review SLA:** How fast should admins respond?
   - **Recommendation:** <1 hour for critical, <24h for medium

4. **Cerbos vs. DB Policies:** Start with Cerbos or DB table?
   - **Recommendation:** DB table for MVP, migrate to Cerbos later

5. **Step-Up Token Lifetime:** 5 minutes too short?
   - **Recommendation:** 5 min default, configurable up to 15 min

---

## 14. References

- [NIST 800-63B: Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [Cerbos Documentation](https://docs.cerbos.dev/)
- [Risk-Based Authentication (Gartner)](https://www.gartner.com/en/documents/3869263)

---

## Revision History

| Version | Date       | Author       | Changes                                          |
| ------- | ---------- | ------------ | ------------------------------------------------ |
| 1.0     | 2025-12-12 | Engineering  | Initial PRD - Risk-Based Adaptive Authentication |
|         |            |              | - Risk to action matrix (DB or Cerbos)           |
|         |            |              | - Adaptive actions (MFA, re-auth, deny, lock)    |
|         |            |              | - Step-up authentication flows                   |
|         |            |              | - Soft session locking                           |
|         |            |              | - Shadow mode for testing                        |
|         |            |              | - Admin override and review queue                |
