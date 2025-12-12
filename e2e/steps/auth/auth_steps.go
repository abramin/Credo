package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
)

// TestContext interface defines the methods needed from the main test context
type TestContext interface {
	POST(path string, body interface{}) error
	GET(path string, headers map[string]string) error
	DELETE(path string, headers map[string]string) error
	GetResponseField(field string) (interface{}, error)
	GetClientID() string
	GetRedirectURI() string
	GetAuthCode() string
	SetAuthCode(code string)
	GetAccessToken() string
	SetAccessToken(token string)
	GetUserID() string
	SetUserID(userID string)
	GetAdminToken() string
	ResponseContains(text string) bool
	GetLastResponseStatus() int
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

	// Admin steps
	ctx.Step(`^I save the user ID from the userinfo response$`, steps.saveUserIDFromUserInfo)
	ctx.Step(`^I delete the user via admin API$`, steps.deleteUserViaAdmin)
	ctx.Step(`^I delete the user via admin API with token "([^"]*)"$`, steps.deleteUserViaAdminWithToken)
	ctx.Step(`^I attempt to delete user with ID "([^"]*)" via admin API$`, steps.deleteSpecificUserViaAdmin)
	ctx.Step(`^I attempt to get user info with the saved access token$`, steps.attemptGetUserInfo)
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

func (s *authSteps) saveUserIDFromUserInfo(ctx context.Context) error {
	userID, err := s.tc.GetResponseField("sub")
	if err != nil {
		return err
	}
	s.tc.SetUserID(userID.(string))
	return nil
}

func (s *authSteps) deleteUserViaAdmin(ctx context.Context) error {
	userID := s.tc.GetUserID()
	if userID == "" {
		return fmt.Errorf("no user ID saved")
	}
	return s.tc.DELETE("/admin/auth/users/"+userID, map[string]string{
		"X-Admin-Token": s.tc.GetAdminToken(),
	})
}

func (s *authSteps) deleteUserViaAdminWithToken(ctx context.Context, token string) error {
	userID := s.tc.GetUserID()
	if userID == "" {
		return fmt.Errorf("no user ID saved")
	}
	return s.tc.DELETE("/admin/auth/users/"+userID, map[string]string{
		"X-Admin-Token": token,
	})
}

func (s *authSteps) deleteSpecificUserViaAdmin(ctx context.Context, userID string) error {
	return s.tc.DELETE("/admin/auth/users/"+userID, map[string]string{
		"X-Admin-Token": s.tc.GetAdminToken(),
	})
}

func (s *authSteps) attemptGetUserInfo(ctx context.Context) error {
	return s.tc.GET("/auth/userinfo", map[string]string{
		"Authorization": "Bearer " + s.tc.GetAccessToken(),
	})
}
