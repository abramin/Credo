package consent

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
	ResponseContains(field string) bool
	GetAccessToken() string
}

// RegisterSteps registers consent-related step definitions
func RegisterSteps(ctx *godog.ScenarioContext, tc TestContext) {
	steps := &consentSteps{tc: tc}

	// Consent management steps
	ctx.Step(`^I am authenticated as "([^"]*)"$`, steps.authenticateAs)
	ctx.Step(`^I grant consent for purposes "([^"]*)"$`, steps.grantConsentForPurposes)
	ctx.Step(`^I revoke consent for purposes "([^"]*)"$`, steps.revokeConsentForPurposes)
	ctx.Step(`^I list my consents$`, steps.listMyConsents)
	ctx.Step(`^I grant consent for purposes "([^"]*)" without authentication$`, steps.grantConsentWithoutAuth)
	ctx.Step(`^I revoke consent for purposes "([^"]*)" without authentication$`, steps.revokeConsentWithoutAuth)
	ctx.Step(`^I POST to "([^"]*)" with empty purposes array$`, steps.postWithEmptyPurposes)
	ctx.Step(`^I wait (\d+) seconds$`, steps.waitSeconds)

	// Consent assertion steps
	ctx.Step(`^the response should contain at least (\d+) consent records$`, steps.responseShouldContainAtLeastNConsents)
	ctx.Step(`^each granted consent should have "([^"]*)" equal to "([^"]*)"$`, steps.eachGrantedConsentShouldHaveField)
	ctx.Step(`^each granted consent should have "([^"]*)"$`, steps.eachGrantedConsentShouldHaveFieldPresent)
	ctx.Step(`^the revoked consent should have "([^"]*)" equal to "([^"]*)"$`, steps.revokedConsentShouldHaveField)
	ctx.Step(`^the revoked consent should have "([^"]*)"$`, steps.revokedConsentShouldHaveFieldPresent)
	ctx.Step(`^the consent for purpose "([^"]*)" should have status "([^"]*)"$`, steps.consentForPurposeShouldHaveStatus)
	ctx.Step(`^the consent should have a new "([^"]*)" timestamp$`, steps.consentShouldHaveNewTimestamp)
}

type consentSteps struct {
	tc TestContext
}

func (s *consentSteps) authenticateAs(ctx context.Context, email string) error {
	// TODO: Implement authentication flow to get access token
	// For now, this is a placeholder that will be implemented
	// when the consent endpoints are fully integrated
	return godog.ErrPending
}

func (s *consentSteps) grantConsentForPurposes(ctx context.Context, purposes string) error {
	body := map[string]interface{}{
		"purposes": strings.Split(purposes, ","),
	}
	return s.tc.POST("/auth/consent", body)
}

func (s *consentSteps) revokeConsentForPurposes(ctx context.Context, purposes string) error {
	body := map[string]interface{}{
		"purposes": strings.Split(purposes, ","),
	}
	return s.tc.POST("/auth/consent/revoke", body)
}

func (s *consentSteps) listMyConsents(ctx context.Context) error {
	return s.tc.GET("/auth/consent", map[string]string{
		"Authorization": "Bearer " + s.tc.GetAccessToken(),
	})
}

func (s *consentSteps) grantConsentWithoutAuth(ctx context.Context, purposes string) error {
	body := map[string]interface{}{
		"purposes": strings.Split(purposes, ","),
	}
	// Make request without Authorization header
	return s.tc.POST("/auth/consent", body)
}

func (s *consentSteps) revokeConsentWithoutAuth(ctx context.Context, purposes string) error {
	body := map[string]interface{}{
		"purposes": strings.Split(purposes, ","),
	}
	// Make request without Authorization header
	return s.tc.POST("/auth/consent/revoke", body)
}

func (s *consentSteps) postWithEmptyPurposes(ctx context.Context, path string) error {
	body := map[string]interface{}{
		"purposes": []string{},
	}
	return s.tc.POST(path, body)
}

func (s *consentSteps) waitSeconds(ctx context.Context, seconds int) error {
	// TODO: Implement wait/sleep functionality
	return godog.ErrPending
}

func (s *consentSteps) responseShouldContainAtLeastNConsents(ctx context.Context, count int) error {
	// TODO: Implement consent count validation
	return godog.ErrPending
}

func (s *consentSteps) eachGrantedConsentShouldHaveField(ctx context.Context, field, value string) error {
	// TODO: Implement field validation for granted consents
	return godog.ErrPending
}

func (s *consentSteps) eachGrantedConsentShouldHaveFieldPresent(ctx context.Context, field string) error {
	// TODO: Implement field presence validation for granted consents
	return godog.ErrPending
}

func (s *consentSteps) revokedConsentShouldHaveField(ctx context.Context, field, value string) error {
	// TODO: Implement field validation for revoked consents
	return godog.ErrPending
}

func (s *consentSteps) revokedConsentShouldHaveFieldPresent(ctx context.Context, field string) error {
	// TODO: Implement field presence validation for revoked consents
	return godog.ErrPending
}

func (s *consentSteps) consentForPurposeShouldHaveStatus(ctx context.Context, purpose, status string) error {
	// TODO: Implement status validation for specific purpose
	return godog.ErrPending
}

func (s *consentSteps) consentShouldHaveNewTimestamp(ctx context.Context, field string) error {
	// TODO: Implement timestamp validation
	return godog.ErrPending
}
