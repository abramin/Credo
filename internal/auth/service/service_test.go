package service

// generate testify test scaffold for Service use gomock for store
//go:generate mockgen -source=service.go -destination=mocks/mocks.go -package=mocks UserStore,SessionStoreimport
import (
	"context"
	"id-gateway/internal/auth/models"
	"id-gateway/internal/auth/service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestService_Authorize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserStore := mocks.NewMockUserStore(ctrl)
	mockSessionStore := mocks.NewMockSessionStore(ctrl)

	flow := NewOIDCFlow()
	service := NewService(flow, mockUserStore, mockSessionStore)

	req := models.AuthorizationRequest{
		ClientID:    "client-123",
		Scopes:      []string{"openid", "profile"},
		RedirectURI: "https://client.app/callback",
		State:       "xyz",
	}

	result, err := service.Authorize(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "todo-session-id", result.SessionID)
	assert.Equal(t, req.RedirectURI, result.RedirectURI)
}
