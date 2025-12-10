package e2e

import (
	"github.com/cucumber/godog"

	"id-gateway/e2e/steps/auth"
	"id-gateway/e2e/steps/common"
	"id-gateway/e2e/steps/consent"
)

// RegisterSteps registers all step definitions from modular packages
func RegisterSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// Register common steps (background, generic requests, assertions)
	common.RegisterSteps(ctx, tc)

	// Register authentication-specific steps
	auth.RegisterSteps(ctx, tc)

	// Register consent-specific steps
	consent.RegisterSteps(ctx, tc)
}
