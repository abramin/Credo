# Authentication Implementation Analysis & Fixes Needed

**Date:** 2025-12-03
**Issue:** Inconsistencies between actual code and documentation

---

## Current State of the Code

### What Actually Exists

**Store Interface** (`internal/auth/store.go`):
```go
type UserStore interface {
    Save(ctx context.Context, user User) error
    FindByID(ctx context.Context, id string) (User, error)
    // ❌ NO FindByEmail method!
}

type SessionStore interface {
    Save(ctx context.Context, session Session) error
    FindByID(ctx context.Context, id string) (Session, error)
}
```

**Service** (`internal/auth/service.go`):
```go
type Service struct {
    flow     *OIDCFlow      // Has stores but doesn't use them!
    users    UserStore
    sessions SessionStore
}

// Thin wrapper - just delegates to OIDCFlow
func (s *Service) Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationResult, error) {
    return s.flow.StartAuthorization(req)
}

// Another thin wrapper
func (s *Service) Token(ctx context.Context, req TokenRequest) (TokenResult, error) {
    return s.flow.ExchangeToken(req)
}
```

**OIDCFlow** (`internal/auth/service.go`):
```go
type OIDCFlow struct{} // No fields!

// Returns hard-coded stub
func (f *OIDCFlow) StartAuthorization(req AuthorizationRequest) (AuthorizationResult, error) {
    return AuthorizationResult{
        SessionID:   "todo-session-id",  // ❌ Not creating real session!
        RedirectURI: req.RedirectURI,
    }, nil
}

// Returns hard-coded stub
func (f *OIDCFlow) ExchangeToken(req TokenRequest) (TokenResult, error) {
    return TokenResult{
        AccessToken: "todo-access",  // ❌ Not real tokens!
        IDToken:     "todo-id",
        ExpiresIn:   3600,
    }, nil
}
```

---

## Problems Identified

### Problem 1: Missing `FindByEmail` Method

**What the PRD says (line 72-80):**
> 2. Check if user exists by email
> 3. If not exists, create new user...
> 4. If exists, retrieve user

**Reality:**
- `UserStore` has NO `FindByEmail` method
- Only has `FindByID`
- Can't implement "find or create by email" flow

**Impact:**
- Handler can't look up users by email
- Can't auto-create users on first login
- PRD-001 implementation is blocked

---

### Problem 2: Confusing Service Architecture

**Current structure:**
```
Handler
  ↓
Service.Authorize()  // Just passes through
  ↓
OIDCFlow.StartAuthorization()  // Returns stub
```

**What's confusing:**
1. **Service has stores but doesn't use them** - Has `users` and `sessions` fields but never calls them
2. **Two similar-named methods** - `Authorize()` vs `StartAuthorization()` - both do nothing useful
3. **OIDCFlow has no state** - Empty struct that returns hard-coded strings
4. **No actual logic** - Everything returns "todo-*" placeholders

**Expected structure (from PRD):**
```
Handler
  ↓
Service.Authorize()  // Should:
  ↓                  // 1. Find/create user
  ↓                  // 2. Create session
  ↓                  // 3. Store in database
  ↓                  // 4. Return real session ID
UserStore + SessionStore
```

---

### Problem 3: PRD Assumes Non-Existent Methods

**PRD-001 Task 1A (line 145):**
> - Create a user if `FindUserByEmail` returns not found

**Tutorial Section 1.4 (line 145):**
> - Create a user if FindUserByEmail returns not found

**Reality:**
- This method doesn't exist
- Implementer will be confused

---

## What Should Be Fixed

### Fix 1: Add `FindByEmail` to UserStore Interface

**Location:** `internal/auth/store.go`

**Add this method:**
```go
type UserStore interface {
    Save(ctx context.Context, user User) error
    FindByID(ctx context.Context, id string) (User, error)
    FindByEmail(ctx context.Context, email string) (User, error) // ← ADD THIS
}
```

**Also add to in-memory implementation** (`internal/auth/store_memory.go`):
```go
func (s *InMemoryUserStore) FindByEmail(ctx context.Context, email string) (User, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for _, user := range s.users {
        if user.Email == email {
            return user, nil
        }
    }
    return User{}, ErrNotFound
}
```

---

### Fix 2: Clarify Service vs OIDCFlow Roles

**Option A: Remove OIDCFlow entirely** (Recommended for simplicity)

Service should directly implement the logic:

```go
type Service struct {
    users    UserStore
    sessions SessionStore
}

func (s *Service) Authorize(ctx context.Context, email, clientID string) (*AuthorizationResult, error) {
    // 1. Find or create user by email
    user, err := s.users.FindByEmail(ctx, email)
    if errors.Is(err, ErrNotFound) {
        // Create new user
        user = User{
            ID:    "user_" + uuid.New().String(),
            Email: email,
            // Extract name from email
        }
        if err := s.users.Save(ctx, user); err != nil {
            return nil, err
        }
    } else if err != nil {
        return nil, err
    }

    // 2. Create session
    session := Session{
        ID:     "sess_" + uuid.New().String(),
        UserID: user.ID,
        Status: "pending_consent",
    }
    if err := s.sessions.Save(ctx, session); err != nil {
        return nil, err
    }

    // 3. Return session ID
    return &AuthorizationResult{
        SessionID: session.ID,
        UserID:    user.ID,
    }, nil
}
```

**Option B: Make OIDCFlow actually do something**

Pass stores to OIDCFlow and move logic there:

```go
type OIDCFlow struct {
    users    UserStore    // ← Add these
    sessions SessionStore
}

func (f *OIDCFlow) StartAuthorization(email, clientID string) (AuthorizationResult, error) {
    // ... actual logic here using f.users and f.sessions
}
```

**Recommendation:** Use Option A. OIDCFlow adds no value currently.

---

### Fix 3: Update PRD-001 to Match Code Structure

**Current PRD says (line 195):**
```go
type AuthService struct {
    users    UserStore
    sessions SessionStore
    now      func() time.Time
}

func (s *AuthService) StartAuthSession(ctx context.Context, email, clientID string) (*Session, *User, error)
```

**Problems:**
1. It's called `AuthService` in PRD but `Service` in code
2. Method `StartAuthSession` doesn't exist - it's called `Authorize`
3. Input signature is wrong - PRD says `(email, clientID)` but code has `(AuthorizationRequest)`
4. Return signature is wrong - PRD returns `(*Session, *User, error)` but code returns `(AuthorizationResult, error)`

**What PRD should say:**
```go
type Service struct {  // Match actual name
    users    UserStore
    sessions SessionStore
}

// Implement this to replace the stub
func (s *Service) Authorize(ctx context.Context, email, clientID string) (*AuthorizationResult, error) {
    // 1. Find or create user by email (requires FindByEmail method)
    // 2. Create session
    // 3. Save session
    // 4. Return AuthorizationResult{ SessionID, UserID }
}
```

---

### Fix 4: Update Tutorial Section 1

**Current tutorial (line 145):**
> - Create a user if `FindUserByEmail` returns not found

**Should say:**
> - Create a user if `users.FindByEmail()` returns `ErrNotFound`
> - Note: You'll need to add `FindByEmail` method to UserStore interface first

**Add prerequisite section:**
```markdown
#### Prerequisite: Add FindByEmail Method

Before implementing the handlers, add this method to UserStore:

Location: `internal/auth/store.go`

```go
type UserStore interface {
    Save(ctx context.Context, user User) error
    FindByID(ctx context.Context, id string) (User, error)
    FindByEmail(ctx context.Context, email string) (User, error) // ADD THIS
}
```

Also implement in `internal/auth/store_memory.go`.
```

---

## Recommended Implementation Path

### Step 1: Fix Store Interface (5 minutes)
1. Add `FindByEmail` to `UserStore` interface
2. Implement in `InMemoryUserStore`

### Step 2: Simplify Service (10 minutes)
1. Remove `OIDCFlow` field from `Service`
2. Move logic directly into `Service.Authorize()`
3. Change signature to accept `email, clientID` directly
4. Actually use the stores!

### Step 3: Update Handler Input (5 minutes)

Handler should parse simple JSON:
```json
{
  "email": "user@example.com",
  "client_id": "demo-client"
}
```

Not the complex `AuthorizationRequest` with RedirectURI/State.

### Step 4: Update Docs (15 minutes)
1. Fix PRD-001 signatures and flow
2. Fix tutorial prerequisites
3. Fix architecture.md if needed

---

## Alternative: Keep Current Structure

If you want to keep OIDCFlow for some reason:

**Make it useful by:**
1. Pass stores to it
2. Move real logic there
3. Make Service a thin wrapper (which it already is)

**But this adds complexity for no benefit.** Simpler to delete OIDCFlow and put logic in Service.

---

## Summary: What's Wrong

| What Docs Say | What Code Has | Problem |
|---------------|---------------|---------|
| `FindByEmail` method | Only `FindByID` | Can't implement "find or create" flow |
| Service uses stores | Service ignores stores | No actual persistence |
| `StartAuthSession` method | `Authorize` method | Name mismatch |
| Returns real session | Returns "todo-session-id" | Not functional |
| Simple email input | Complex AuthorizationRequest | Over-engineered |
| Service has logic | OIDCFlow has stubs | Confusing layers |

---

## Action Items

**For Code:**
- [ ] Add `FindByEmail` to UserStore interface
- [ ] Implement `FindByEmail` in InMemoryUserStore
- [ ] Move real logic into Service.Authorize()
- [ ] Delete OIDCFlow (or make it actually useful)
- [ ] Change Authorize signature to accept simple params

**For Docs:**
- [ ] Update PRD-001 method signatures
- [ ] Update PRD-001 to show FindByEmail addition
- [ ] Update tutorial prerequisites
- [ ] Update architecture.md service descriptions
- [ ] Add "Current Code Issues" section to tutorial

---

## Questions for You

1. **Do you want to keep OIDCFlow?** If so, why? It currently adds no value.

2. **Should handlers accept email directly** or parse complex AuthorizationRequest?
   - Simple: `{"email": "...", "client_id": "..."}`
   - Complex: `{"client_id": "...", "scopes": [...], "redirect_uri": "...", "state": "..."}`

3. **Should I create fixed versions of PRD-001 and Tutorial**, or would you prefer to fix them based on this analysis?

Let me know and I can update the docs to match a working implementation!
