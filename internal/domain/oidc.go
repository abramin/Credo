package domain

// OIDCFlow holds the business rules for the mock OIDC login and consent dance.
// Storage and HTTP concerns live in other layers; this stays pure.
type OIDCFlow struct{}

func NewOIDCFlow() *OIDCFlow {
	return &OIDCFlow{}
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

func (f *OIDCFlow) StartAuthorization(req AuthorizationRequest) (AuthorizationResult, error) {
	return AuthorizationResult{
		SessionID:   "todo-session-id",
		RedirectURI: req.RedirectURI,
	}, nil
}

func (f *OIDCFlow) RecordConsent(req ConsentRequest) (ConsentResult, error) {
	return ConsentResult{
		SessionID: req.SessionID,
		Approved:  req.Approved,
	}, nil
}

func (f *OIDCFlow) ExchangeToken(req TokenRequest) (TokenResult, error) {
	return TokenResult{
		AccessToken: "todo-access",
		IDToken:     "todo-id",
		ExpiresIn:   3600,
	}, nil
}
