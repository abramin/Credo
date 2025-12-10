package auth

import (
	"context"
	"strings"

	"github.com/cucumber/godog"
)

// TestContext interface defines the methods needed from the main test context
type TestContext interface {
	POST(path string, body interface{}) error
	GET(path string, headers map[string]string) error
	GetResponseField(field string) (interface{}, error)
	GetClientID() string
	GetRedirectURI() string
	GetAuthCode() string
	SetAuthCode(code string)
	GetAccessToken() string
	SetAccessToken(token string)
}

// RegisterSteps registers authentication-related step definitions
func RegisterSteps(ctx *godog.ScenarioContext, tc TestContext) {
	steps := &authSteps{tc: tc}

	// Authorization steps
	ctx.Step(`^I initiate authorization with email "([^"]*)" and scopes "([^"]*)"$`, steps.initiateAuthorization)
	ctx.Step(`^I save the authorization code$`, steps.saveAuthorizationCode)
	ctx.Step(`^I exchange the authorization code for tokens$`, steps.exchangeCodeForTokens)
	ctx.Step(`^I exchange invalid authorization code "([^"]*)"$`, steps.exchangeInvalidCode)
	ctx.Step(`^I attempt to reuse the same authorization code$`, steps.reuseAuthorizationCode)
	ctx.Step(`^I request user info with the access token$`, steps.requestUserInfo)

	// Validation steps
	ctx.Step(`^I POST to "([^"]*)" with invalid email "([^"]*)"$`, steps.postWithInvalidEmail)
	ctx.Step(`^I POST to "([^"]*)" with grant_type "([^"]*)"$`, steps.postWithGrantType)
	ctx.Step(`^I GET "([^"]*)" with invalid token "([^"]*)"$`, steps.getWithInvalidToken)
}

type authSteps struct {
	tc TestContext
}

func (s *authSteps) initiateAuthorization(ctx context.Context, email, scopes string) error {
	body := map[string]interface{}{
		"email":        email,
		"client_id":    s.tc.GetClientID(),
		"scopes":       strings.Split(scopes, ","),
		"redirect_uri": s.tc.GetRedirectURI(),
		"state":        "test-state-123",
	}
	return s.tc.POST("/auth/authorize", body)
}

func (s *authSteps) saveAuthorizationCode(ctx context.Context) error {
	code, err := s.tc.GetResponseField("code")
	if err != nil {
		return err
	}
	s.tc.SetAuthCode(code.(string))
	return nil
}

func (s *authSteps) exchangeCodeForTokens(ctx context.Context) error {
	body := map[string]interface{}{
		"grant_type":   "authorization_code",
		"code":         s.tc.GetAuthCode(),
		"redirect_uri": s.tc.GetRedirectURI(),
		"client_id":    s.tc.GetClientID(),
	}
	return s.tc.POST("/auth/token", body)
}

func (s *authSteps) exchangeInvalidCode(ctx context.Context, code string) error {
	body := map[string]interface{}{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": s.tc.GetRedirectURI(),
		"client_id":    s.tc.GetClientID(),
	}
	return s.tc.POST("/auth/token", body)
}

func (s *authSteps) reuseAuthorizationCode(ctx context.Context) error {
	return s.exchangeCodeForTokens(ctx)
}

func (s *authSteps) postWithInvalidEmail(ctx context.Context, path, email string) error {
	body := map[string]interface{}{
		"email":        email,
		"client_id":    s.tc.GetClientID(),
		"scopes":       []string{"openid"},
		"redirect_uri": s.tc.GetRedirectURI(),
	}
	return s.tc.POST(path, body)
}

func (s *authSteps) postWithGrantType(ctx context.Context, path, grantType string) error {
	body := map[string]interface{}{
		"grant_type":   grantType,
		"code":         "some-code",
		"redirect_uri": s.tc.GetRedirectURI(),
		"client_id":    s.tc.GetClientID(),
	}
	return s.tc.POST(path, body)
}

func (s *authSteps) getWithInvalidToken(ctx context.Context, path, token string) error {
	return s.tc.GET(path, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

func (s *authSteps) requestUserInfo(ctx context.Context) error {
	accessToken, err := s.tc.GetResponseField("access_token")
	if err != nil {
		return err
	}
	s.tc.SetAccessToken(accessToken.(string))

	return s.tc.GET("/auth/userinfo", map[string]string{
		"Authorization": "Bearer " + s.tc.GetAccessToken(),
	})
}
