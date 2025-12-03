package domain

// VCLifecycle owns verifiable credential issuance and verification rules.
// Anything related to signing, persistence, or registries is orchestrated by
// callers so this type can remain dependency-free.
type VCLifecycle struct{}

func NewVCLifecycle() *VCLifecycle {
	return &VCLifecycle{}
}

type VCClaims map[string]interface{}

type IssueVCRequest struct {
	SubjectID string
	Claims    VCClaims
}

type IssueVCResult struct {
	ID         string
	Credential VCClaims
}

type VerifyVCRequest struct {
	Credential VCClaims
}

type VerifyVCResult struct {
	Valid bool
}

func (l *VCLifecycle) Issue(req IssueVCRequest) (IssueVCResult, error) {
	return IssueVCResult{
		ID:         "todo-credential-id",
		Credential: MinimizeClaims(req.Claims),
	}, nil
}

func (l *VCLifecycle) Verify(req VerifyVCRequest) (VerifyVCResult, error) {
	return VerifyVCResult{Valid: true}, nil
}

// MinimizeClaims removes raw PII while keeping derived assertions. This is a
// placeholder until concrete claim schemas are defined.
func MinimizeClaims(claims VCClaims) VCClaims {
	out := VCClaims{}
	for k, v := range claims {
		// Drop known PII keys.
		if k == "full_name" || k == "national_id" {
			continue
		}
		out[k] = v
	}
	return out
}
