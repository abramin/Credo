# PRD-008: Automated GDPR/CCPA Compliance Checking

**Status:** Not Started
**Priority:** P1 (High - Practical Showcase)
**Owner:** Engineering Team
**Dependencies:** PRD-006 (Audit), PRD-007 (Data Rights) complete
**Last Updated:** 2025-12-06

---

## 1. Overview

### Problem Statement
Current implementation handles GDPR data rights reactively (export/delete on demand), but doesn't proactively check if the system is compliant with privacy regulations. Organizations need automated verification that:
- Data retention limits are respected
- Consent is obtained before data processing
- User rights requests are handled within legal timeframes
- PII is properly minimized in regulated mode
- Audit logs capture required information

### Goals
- Define compliance policies as code (data retention, consent requirements, response timeframes)
- Implement automated compliance checker that validates system state
- Provide compliance dashboard showing adherence to policies
- Alert on policy violations before they become legal issues
- Generate compliance reports for auditors
- Support both GDPR (EU) and CCPA (California) requirements

### Non-Goals
- Legal advice or interpretation of regulations
- Dynamic policy updates via UI (policies are code for MVP)
- Multi-jurisdictional policy management
- Integration with external compliance tools
- Privacy impact assessments (PIA)
- Data processing agreements (DPA) generation

---

## 2. User Stories

**As a** compliance officer
**I want to** see a real-time compliance dashboard
**So that** I can verify we're meeting GDPR/CCPA requirements

**As a** data protection officer
**I want to** receive alerts when policies are violated
**So that** I can fix issues before regulatory audits

**As an** engineering manager
**I want to** automated compliance checks in CI/CD
**So that** non-compliant changes are caught before production

**As a** developer
**I want to** clear policy definitions
**So that** I know what constraints to follow when building features

---

## 3. Compliance Policies

### Policy Categories

#### PC-1: Data Retention
**GDPR Article 5(1)(e):** Storage limitation - Data kept no longer than necessary

**Policies:**
- **Policy 1.1:** User data deleted within 30 days of account deletion request (GDPR Article 17 response time)
- **Policy 1.2:** Audit logs retained for 6 years (financial regulation requirement)
- **Policy 1.3:** Session data deleted after 30 days of inactivity
- **Policy 1.4:** Cached registry data refreshed/deleted after 24 hours

#### PC-2: Consent Requirements
**GDPR Article 6:** Lawfulness of processing requires consent

**Policies:**
- **Policy 2.1:** Registry lookups require explicit consent
- **Policy 2.2:** VC issuance requires explicit consent
- **Policy 2.3:** Decision evaluation requires consent for data processing
- **Policy 2.4:** Consent can be withdrawn at any time

#### PC-3: User Rights Response Time
**GDPR Article 12(3):** Response within 1 month

**Policies:**
- **Policy 3.1:** Data export requests fulfilled within 72 hours (stricter than GDPR)
- **Policy 3.2:** Data deletion requests completed within 24 hours (stricter than GDPR)
- **Policy 3.3:** Consent revocation takes effect immediately

#### PC-4: Data Minimization
**GDPR Article 5(1)(c):** Adequate, relevant, and limited to what is necessary

**Policies:**
- **Policy 4.1:** PII removed from logs in regulated mode
- **Policy 4.2:** VC claims minimized to boolean flags in regulated mode
- **Policy 4.3:** Decision inputs use derived attributes, not raw PII

#### PC-5: Audit Trail Completeness
**GDPR Article 30:** Records of processing activities

**Policies:**
- **Policy 5.1:** All consent grants/revocations logged
- **Policy 5.2:** All data access logged with purpose
- **Policy 5.3:** All data exports/deletions logged
- **Policy 5.4:** Logs include: who, what, when, why, decision

---

## 4. Functional Requirements

### FR-1: Define Compliance Policies
**Location:** `internal/compliance/policies.go` (new file)

**Description:** Policies defined as Go code with clear structure.

**Example Policy Definition:**
```go
type Policy struct {
    ID          string
    Category    string
    Name        string
    Description string
    Regulation  string // "GDPR", "CCPA", "Both"
    Check       func(ctx context.Context, checker *Checker) PolicyResult
}

var DataRetentionPolicies = []Policy{
    {
        ID:          "retention-1.1",
        Name:        "User Data Deletion Timeliness",
        Description: "User data deleted within 30 days of deletion request",
        Regulation:  "GDPR",
        Check:       checkUserDeletionTimeliness,
    },
    {
        ID:          "retention-1.2",
        Name:        "Audit Log Retention",
        Description: "Audit logs retained for at least 6 years",
        Regulation:  "Both",
        Check:       checkAuditRetention,
    },
}
```

---

### FR-2: Run Compliance Check
**Endpoint:** `POST /compliance/check`

**Description:** Run all compliance policies and return results.

**Input:**
```json
{
  "policy_ids": ["retention-1.1", "consent-2.1"], // Optional: check specific policies
  "regulation": "GDPR" // Optional: filter by regulation
}
```

**Output (Success - 200):**
```json
{
  "check_id": "check_abc123",
  "checked_at": "2025-12-06T10:00:00Z",
  "regulation": "GDPR",
  "summary": {
    "total": 12,
    "passed": 10,
    "failed": 2,
    "warnings": 3
  },
  "results": [
    {
      "policy_id": "retention-1.1",
      "name": "User Data Deletion Timeliness",
      "status": "pass",
      "message": "All deletion requests completed within 30 days"
    },
    {
      "policy_id": "consent-2.1",
      "name": "Registry Lookup Consent",
      "status": "fail",
      "message": "Found 3 registry lookups without prior consent",
      "violations": [
        {
          "user_id": "user_123",
          "timestamp": "2025-12-05T14:00:00Z",
          "details": "Registry lookup performed without consent"
        }
      ]
    },
    {
      "policy_id": "audit-5.1",
      "name": "Consent Change Logging",
      "status": "warning",
      "message": "1 consent grant event missing purpose field"
    }
  ]
}
```

**Business Logic:**
1. Parse request (specific policies or all)
2. Load relevant policies from definitions
3. Run each policy check function
4. Collect results (pass/fail/warning)
5. Generate summary statistics
6. Return structured results

**Error Cases:**
- 400 Bad Request: Invalid policy ID
- 401 Unauthorized: Not admin
- 500 Internal Server Error: Check execution failed

---

### FR-3: Get Compliance Dashboard
**Endpoint:** `GET /compliance/dashboard`

**Description:** Return compliance overview with historical trends.

**Output (Success - 200):**
```json
{
  "current_status": {
    "compliant": true,
    "score": 83,
    "last_check": "2025-12-06T10:00:00Z",
    "critical_failures": 0,
    "warnings": 3
  },
  "by_category": {
    "data_retention": {"passed": 4, "failed": 0, "warnings": 1},
    "consent": {"passed": 3, "failed": 0, "warnings": 2},
    "user_rights": {"passed": 3, "failed": 0, "warnings": 0},
    "data_minimization": {"passed": 2, "failed": 0, "warnings": 0},
    "audit_trail": {"passed": 3, "failed": 0, "warnings": 0}
  },
  "recent_violations": [
    {
      "policy_id": "audit-5.1",
      "timestamp": "2025-12-06T09:30:00Z",
      "severity": "warning",
      "message": "Missing purpose in audit event"
    }
  ],
  "trends": {
    "compliance_score_7d": [78, 80, 82, 81, 83, 83, 83],
    "violations_7d": [2, 1, 1, 0, 0, 1, 0]
  }
}
```

---

### FR-4: Generate Compliance Report
**Endpoint:** `GET /compliance/report`

**Description:** Generate detailed compliance report for auditors.

**Query Params:**
- `format`: "json" | "pdf" (default: json)
- `regulation`: "GDPR" | "CCPA" | "Both"
- `from`: Start date (ISO 8601)
- `to`: End date (ISO 8601)

**Output (JSON format):**
```json
{
  "report_id": "report_abc123",
  "generated_at": "2025-12-06T10:00:00Z",
  "period": {
    "from": "2025-11-01T00:00:00Z",
    "to": "2025-12-06T23:59:59Z"
  },
  "regulation": "GDPR",
  "executive_summary": {
    "overall_compliance": "95%",
    "critical_issues": 0,
    "resolved_issues": 12,
    "open_warnings": 3
  },
  "policy_results": [...],
  "user_rights_summary": {
    "data_export_requests": 23,
    "avg_response_time": "4.2 hours",
    "data_deletion_requests": 5,
    "avg_deletion_time": "12 hours"
  },
  "consent_summary": {
    "consent_grants": 145,
    "consent_revocations": 12,
    "processing_with_consent": "100%"
  },
  "recommendations": [
    "Consider reducing audit log retention to 3 years for non-financial data",
    "Improve purpose documentation in 3 audit events"
  ]
}
```

---

### FR-5: Policy Violation Alerts (Internal)
**Background Process** (no endpoint)

**Description:** Continuously monitor for policy violations and alert administrators.

**Alert Triggers:**
- Critical policy failure (e.g., processing without consent)
- Data retention limit exceeded
- User rights request approaching deadline
- Missing audit log entries

**Alert Channels (MVP):**
- Log to stderr with CRITICAL level
- Store in violation log table
- (Future: Email, Slack, PagerDuty)

---

## 5. Technical Requirements

### TR-1: Data Models

**Location:** `internal/compliance/models.go` (new file)

```go
type PolicyResult struct {
    PolicyID   string
    Name       string
    Status     PolicyStatus // "pass", "fail", "warning"
    Message    string
    Violations []Violation
    CheckedAt  time.Time
}

type PolicyStatus string

const (
    PolicyPass    PolicyStatus = "pass"
    PolicyFail    PolicyStatus = "fail"
    PolicyWarning PolicyStatus = "warning"
)

type Violation struct {
    UserID    string
    Timestamp time.Time
    Details   string
    Severity  string // "critical", "high", "medium", "low"
}

type ComplianceCheck struct {
    ID         string
    Regulation string
    CheckedAt  time.Time
    Results    []PolicyResult
    Summary    CheckSummary
}

type CheckSummary struct {
    Total    int
    Passed   int
    Failed   int
    Warnings int
}
```

### TR-2: Policy Checker Service

**Location:** `internal/compliance/checker.go` (new file)

```go
type Checker struct {
    userStore     auth.UserStore
    sessionStore  auth.SessionStore
    consentStore  consent.Store
    auditStore    audit.Store
    vcStore       vc.Store
    regulatedMode bool
}

func (c *Checker) CheckAllPolicies(ctx context.Context, regulation string) (*ComplianceCheck, error) {
    policies := LoadPolicies(regulation)
    results := []PolicyResult{}
    
    for _, policy := range policies {
        result := policy.Check(ctx, c)
        results = append(results, result)
    }
    
    return &ComplianceCheck{
        ID:         uuid.New().String(),
        Regulation: regulation,
        CheckedAt:  time.Now(),
        Results:    results,
        Summary:    summarizeResults(results),
    }, nil
}
```

### TR-3: Policy Check Functions

**Location:** `internal/compliance/checks.go` (new file)

**Example Checks:**

```go
// Check: User data deleted within 30 days of request
func checkUserDeletionTimeliness(ctx context.Context, c *Checker) PolicyResult {
    // Query audit log for data_deletion_requested events
    deletionEvents, _ := c.auditStore.ListByAction(ctx, "data_deletion_requested")
    
    violations := []Violation{}
    for _, event := range deletionEvents {
        // Check if user still exists after 30 days
        if time.Since(event.Timestamp) > 30*24*time.Hour {
            user, _ := c.userStore.GetByID(ctx, event.UserID)
            if user != nil {
                violations = append(violations, Violation{
                    UserID:    event.UserID,
                    Timestamp: event.Timestamp,
                    Details:   "User data not deleted within 30 days",
                    Severity:  "high",
                })
            }
        }
    }
    
    status := PolicyPass
    message := "All deletion requests completed timely"
    if len(violations) > 0 {
        status = PolicyFail
        message = fmt.Sprintf("%d deletion requests exceeded 30 days", len(violations))
    }
    
    return PolicyResult{
        PolicyID:   "retention-1.1",
        Name:       "User Data Deletion Timeliness",
        Status:     status,
        Message:    message,
        Violations: violations,
        CheckedAt:  time.Now(),
    }
}

// Check: Registry lookups have prior consent
func checkRegistryConsentCompliance(ctx context.Context, c *Checker) PolicyResult {
    // Get all registry lookup events
    registryEvents, _ := c.auditStore.ListByAction(ctx, "registry_citizen_checked")
    
    violations := []Violation{}
    for _, event := range registryEvents {
        // Check if consent existed before lookup
        consents, _ := c.consentStore.ListByUser(ctx, event.UserID)
        hasConsent := false
        for _, consent := range consents {
            if consent.Purpose == "registry_check" && consent.GrantedAt.Before(event.Timestamp) {
                hasConsent = true
                break
            }
        }
        
        if !hasConsent {
            violations = append(violations, Violation{
                UserID:    event.UserID,
                Timestamp: event.Timestamp,
                Details:   "Registry lookup without prior consent",
                Severity:  "critical",
            })
        }
    }
    
    status := PolicyPass
    if len(violations) > 0 {
        status = PolicyFail
    }
    
    return PolicyResult{
        PolicyID:   "consent-2.1",
        Status:     status,
        Message:    fmt.Sprintf("Found %d consent violations", len(violations)),
        Violations: violations,
        CheckedAt:  time.Now(),
    }
}

// Check: Audit logs complete
func checkAuditCompleteness(ctx context.Context, c *Checker) PolicyResult {
    events, _ := c.auditStore.ListAll(ctx, 1000, 0)
    
    warnings := []Violation{}
    for _, event := range events {
        // Check required fields present
        if event.Purpose == "" {
            warnings = append(warnings, Violation{
                UserID:    event.UserID,
                Timestamp: event.Timestamp,
                Details:   fmt.Sprintf("Event %s missing purpose field", event.ID),
                Severity:  "low",
            })
        }
        if event.Decision == "" && requiresDecision(event.Action) {
            warnings = append(warnings, Violation{
                UserID:    event.UserID,
                Timestamp: event.Timestamp,
                Details:   fmt.Sprintf("Event %s missing decision field", event.ID),
                Severity:  "medium",
            })
        }
    }
    
    status := PolicyPass
    if len(warnings) > 0 {
        status = PolicyWarning
    }
    
    return PolicyResult{
        PolicyID:   "audit-5.1",
        Status:     status,
        Message:    fmt.Sprintf("%d audit events need improvement", len(warnings)),
        Violations: warnings,
        CheckedAt:  time.Now(),
    }
}
```

### TR-4: HTTP Handlers

**Location:** `internal/transport/http/handlers_compliance.go` (new file)

```go
func (h *Handler) handleComplianceCheck(w http.ResponseWriter, r *http.Request)
func (h *Handler) handleComplianceDashboard(w http.ResponseWriter, r *http.Request)
func (h *Handler) handleComplianceReport(w http.ResponseWriter, r *http.Request)
```

---

## 6. Implementation Steps

### Phase 1: Policy Definitions (2-3 hours)
1. Create `internal/compliance/` package
2. Define policy data models
3. Write 10-12 core policies covering all categories
4. Document each policy with regulation references

### Phase 2: Checker Service (3-4 hours)
1. Implement Checker service with store dependencies
2. Write policy check functions (one per policy)
3. Test each check function independently

### Phase 3: Compliance Check Endpoint (2 hours)
1. Implement handleComplianceCheck
2. Run all policies and collect results
3. Return structured response

### Phase 4: Dashboard & Reporting (3-4 hours)
1. Implement handleComplianceDashboard
2. Calculate compliance score and trends
3. Implement handleComplianceReport
4. Generate detailed compliance reports

### Phase 5: Violation Monitoring (2 hours)
1. Background goroutine running periodic checks
2. Alert logging for critical failures
3. Store violations for historical tracking

### Phase 6: Testing & Documentation (3-4 hours)
1. Unit tests for each policy check
2. Integration tests for full compliance flow
3. Document all policies and their checks
4. Create compliance guide for developers

---

## 7. Acceptance Criteria

- [ ] 10+ compliance policies defined across all categories
- [ ] All policies have automated check functions
- [ ] Compliance check endpoint returns structured results
- [ ] Dashboard shows real-time compliance status
- [ ] Reports can be generated for specific time periods
- [ ] Critical violations trigger alerts
- [ ] All policies reference specific GDPR/CCPA articles
- [ ] Policies detect actual violations (test with intentional violations)
- [ ] Documentation explains each policy clearly
- [ ] Performance acceptable (checks run in <5 seconds)

---

## 8. Testing

### Unit Tests
```go
func TestDataDeletionTimeliness(t *testing.T) {
    // Create deletion request event 31 days ago
    // User still exists
    // Run check
    // Expect: PolicyFail with violation
}

func TestConsentCompliance(t *testing.T) {
    // Registry lookup event
    // No prior consent
    // Run check
    // Expect: PolicyFail with critical violation
}
```

### Integration Tests
```bash
# Perform operations, some compliant, some not

# Check consent violation (lookup without consent)
# Don't grant consent
curl -X POST /registry/citizen -H "Authorization: Bearer $TOKEN" \
  -d '{"national_id": "123"}'
# Expected: 403 (in practice this should fail, but let's say it goes through for test)

# Run compliance check
curl -X POST http://localhost:8080/compliance/check \
  -d '{"regulation": "GDPR"}'

# Expected: Some policies pass, consent-2.1 fails

# View dashboard
curl http://localhost:8080/compliance/dashboard

# Expected: Overall compliance score, violations listed

# Generate report
curl "http://localhost:8080/compliance/report?regulation=GDPR&from=2025-11-01"

# Expected: Detailed compliance report with violations
```

---

## 9. Compliance Score Calculation

### Scoring Algorithm
```
Base Score = 100

For each policy:
  - Pass: +0 points (neutral)
  - Warning: -2 points
  - Fail (low severity): -5 points
  - Fail (medium severity): -10 points
  - Fail (high severity): -20 points
  - Fail (critical severity): -30 points

Minimum Score = 0
Compliance Status:
  - 95-100: Excellent
  - 85-94: Good
  - 70-84: Needs Improvement
  - <70: Non-Compliant
```

---

## 10. Dashboard UI (Optional Frontend)

**Location:** `frontend/public/compliance.html` (new file)

**Features:**
- Real-time compliance score gauge
- Policy status cards (green/yellow/red)
- Recent violations timeline
- Compliance score trend chart
- Quick report generation
- Policy documentation links

**Tech Stack:**
- HTML/CSS/JavaScript
- Chart.js for visualizations
- Fetch API for backend calls

---

## 11. Future Enhancements

- Dynamic policy definitions (JSON/YAML config files)
- Policy versioning and history
- Custom policy builder UI
- Integration with external compliance tools
- Automated remediation actions
- Multi-region policy management
- Data Processing Agreements (DPA) generation
- Privacy Impact Assessment (PIA) templates
- Real-time alerting (Slack, PagerDuty)
- Compliance as Code in CI/CD

---

## 12. References

- [GDPR Full Text](https://gdpr-info.eu/)
- [CCPA Full Text](https://oag.ca.gov/privacy/ccpa)
- [GDPR Article 5: Principles](https://gdpr-info.eu/art-5-gdpr/)
- [GDPR Article 30: Records of Processing](https://gdpr-info.eu/art-30-gdpr/)
- Existing Code: PRD-006 (Audit), PRD-007 (Data Rights)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-06 | Engineering Team | Initial PRD |
