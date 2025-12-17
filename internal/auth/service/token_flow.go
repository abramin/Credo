package service

import (
	"context"
	"time"

	"credo/internal/auth/models"
	dErrors "credo/pkg/domain-errors"
)

// tokenFlowTxParams captures the inputs for token transaction execution.
// Both authorization code exchange and refresh token flows use this structure.
type tokenFlowTxParams struct {
	Session      *models.Session
	TokenContext *tokenContext
	Now          time.Time
	// ActivateOnFirstUse is true for code exchange (activates pending sessions),
	// false for refresh (updates LastRefreshedAt instead).
	ActivateOnFirstUse bool
}

// tokenFlowTxResult holds the outputs from a successful token transaction.
type tokenFlowTxResult struct {
	Session   *models.Session
	Artifacts *tokenArtifacts
}

// executeTokenFlowTx runs the common transactional portion of token issuance.
// It handles device binding, token generation, session advancement, and refresh token creation.
func (s *Service) executeTokenFlowTx(
	ctx context.Context,
	stores TxAuthStores,
	params tokenFlowTxParams,
) (*tokenFlowTxResult, error) {
	mutableSession := *params.Session
	s.applyDeviceBinding(ctx, &mutableSession)
	mutableSession.LastSeenAt = params.Now

	activate := false
	if params.ActivateOnFirstUse {
		// Code exchange: activate session if pending consent
		if mutableSession.Status == string(models.SessionStatusPendingConsent) {
			mutableSession.Status = string(models.SessionStatusActive)
			activate = true
		}
	} else {
		// Refresh: update refresh timestamp
		mutableSession.LastRefreshedAt = &params.Now
	}

	artifacts, err := s.generateTokenArtifacts(&mutableSession)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to generate tokens")
	}

	// Advance session state based on flow type
	var session *models.Session
	clientID := params.TokenContext.Client.ID.String()

	if params.ActivateOnFirstUse {
		session, err = stores.Sessions.AdvanceLastSeen(
			ctx,
			params.Session.ID,
			clientID,
			params.Now,
			artifacts.accessTokenJTI,
			activate,
			mutableSession.DeviceID,
			mutableSession.DeviceFingerprintHash,
		)
	} else {
		session, err = stores.Sessions.AdvanceLastRefreshed(
			ctx,
			params.Session.ID,
			clientID,
			params.Now,
			artifacts.accessTokenJTI,
			mutableSession.DeviceID,
			mutableSession.DeviceFingerprintHash,
		)
	}
	if err != nil {
		return nil, err
	}

	if err := stores.RefreshTokens.Create(ctx, artifacts.refreshRecord); err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to create refresh token")
	}

	return &tokenFlowTxResult{
		Session:   session,
		Artifacts: artifacts,
	}, nil
}
