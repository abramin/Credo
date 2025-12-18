# PRD-025: Developer Sandbox & Testing

**Status:** Not Started
**Priority:** P2 (Medium)
**Owner:** Engineering Team
**Dependencies:** PRD-001-005, PRD-014
**Last Updated:** 2025-12-18

## 1. Overview

### Problem Statement
High friction for partners to integrate - no test environment or test data.

### Goals
- Sandbox environment (separate from prod)
- Test mode flag (no real charges, no real emails)
- Mock registry responses
- Test user generation
- API explorer / Postman collections
- Interactive documentation (try-it-out)
- Developer dashboard (API keys, usage stats)

## 2. Functional Requirements

### FR-1: Sandbox Environment
**URL:** `https://sandbox.credo.dev`
**Database:** Separate from production
**No real notifications:** Emails/SMS mocked

### FR-2: Test Data Generator
**Endpoint:** `POST /sandbox/users/generate`
**Output:** Creates test users with various states

### FR-3: Mock Registry
**Sandbox registry returns:** Deterministic responses for testing

### FR-4: API Explorer
**Swagger UI:** Embedded in docs
**Try it out:** Execute requests from docs

## 3. Acceptance Criteria
- [ ] Sandbox environment deployed
- [ ] Test users generated via API
- [ ] Mock registry provides predictable data
- [ ] API explorer functional
- [ ] Developer dashboard shows usage
- [ ] DSA katas included: LRU cache and rate limiter variants behind feature flags with fixtures to compare behaviors
- [ ] SQL anti-pattern lab: migrations include intentional N+1/index issues; fixes validated via EXPLAIN
- [ ] Security defaults: deny-by-default network egress, sealed secrets, and mutation testing for input validation paths

## Revision History
| Version | Date       | Author       | Changes                                                              |
| ------- | ---------- | ------------ | -------------------------------------------------------------------- |
| 1.1     | 2025-12-18 | Security Eng | Added DSA katas, SQL anti-pattern lab, and security defaults         |
| 1.0     | 2025-12-12 | Product Team | Initial PRD                                                          |
