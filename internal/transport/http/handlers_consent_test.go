package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	consentModel "id-gateway/internal/consent/models"
	"id-gateway/internal/platform/metrics"
	"id-gateway/internal/platform/middleware"
	"id-gateway/internal/transport/http/mocks"
)

//go:generate mockgen -source=handlers_consent.go -destination=mocks/consent-mocks.go -package=mocks ConsentService

func TestConsentHandler_handleGrantConsent_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConsent := mocks.NewMockConsentService(ctrl)
	mockConsent.EXPECT().
		Grant(gomock.Any(), "user123", consentModel.ConsentPurposeLogin, 24*time.Hour).
		Return(nil).
		Times(1)

	handler := &ConsentHandler{
		logger:     nil,
		consent:    mockConsent,
		metrics:    &metrics.Metrics{},
		consentTTL: 24 * time.Hour,
	}

	grantReq := consentModel.GrantConsentRequest{
		Purposes: []consentModel.ConsentPurpose{consentModel.ConsentPurposeLogin},
	}

	body, err := json.Marshal(grantReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/consent", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "user123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.handleGrantConsent(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

}
