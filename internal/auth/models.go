package auth

// User captures the primary identity tracked by the gateway. Storage of the
// actual user record lives behind the UserStore interface.
type User struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Verified  bool
}

// Session models an authorization session.
type Session struct {
	ID             string
	UserID         string
	RequestedScope []string
	Status         string
}

type AuthorizationRequest struct {
	ClientID    string
	Scopes      []string
	RedirectURI string
	State       string
}

type AuthorizationResult struct {
	SessionID   string
	RedirectURI string
}

type ConsentRequest struct {
	SessionID string
	Approved  bool
}

type ConsentResult struct {
	SessionID string
	Approved  bool
}

type TokenRequest struct {
	SessionID string
	Code      string
}

type TokenResult struct {
	AccessToken string
	IDToken     string
	ExpiresIn   int
}
