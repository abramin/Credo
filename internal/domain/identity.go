package domain

import "time"

// User captures the primary identity tracked by the gateway. Storage of the
// actual user record lives in the storage layer.
type User struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Verified  bool
}

// Session models an OIDC authorization session.
type Session struct {
	ID             string
	UserID         string
	RequestedScope []string
	Status         string
}

// DerivedIdentityFromCitizen strips PII while producing attributes required for
// decisions in regulated mode.
func DerivedIdentityFromCitizen(user User, citizen CitizenRecord) DerivedIdentity {
	isOver18 := deriveIsOver18(citizen.DateOfBirth)
	return DerivedIdentity{
		PseudonymousID: user.ID, // treat as pseudonymous identifier; avoid emails/names.
		IsOver18:       isOver18,
		CitizenValid:   citizen.Valid,
	}
}

func deriveIsOver18(dob string) bool {
	if dob == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return false
	}
	years := time.Since(t).Hours() / 24 / 365.25
	return years >= 18
}
