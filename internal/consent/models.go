package consent

import (
	"time"

	pkgerrors "id-gateway/pkg/http-errors"
)

// ConsentPurpose labels why data is processed. Purpose binding allows selective
// revocation without affecting other flows.
type ConsentPurpose string

const (
	ConsentPurposeLogin      ConsentPurpose = "login"
	ConsentPurposeRegistry   ConsentPurpose = "registry_check"
	ConsentPurposeVCIssuance ConsentPurpose = "vc_issuance"
)

// ConsentRecord captures a user's decision for a specific purpose.
type ConsentRecord struct {
	UserID    string
	Purpose   ConsentPurpose
	GrantedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// IsActive returns true when consent is currently valid.
func (c ConsentRecord) IsActive(now time.Time) bool {
	if c.RevokedAt != nil && c.RevokedAt.Before(now) {
		return false
	}
	return now.Before(c.ExpiresAt) || c.ExpiresAt.IsZero()
}

// EnsureConsent enforces that consent exists and is active for the given purpose.
func EnsureConsent(consents []ConsentRecord, purpose ConsentPurpose, now time.Time) error {
	for _, c := range consents {
		if c.Purpose == purpose && c.IsActive(now) {
			return nil
		}
	}
	return pkgerrors.New(pkgerrors.CodeMissingConsent, "consent not granted for required purpose")
}
