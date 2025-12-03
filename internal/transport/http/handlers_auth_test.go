package httptransport

import (
	"context"
	"encoding/json"
	"id-gateway/internal/transport/http/mocks"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authModel "id-gateway/internal/auth/models"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=handlers_auth.go -destination=mocks/auth-mocks.go -package=mocks AuthService
type AuthHandlerSuite struct {
	suite.Suite
	handler     *AuthHandler
	ctx         context.Context
	router      http.ServeMux
	mockService *mocks.MockAuthService
	ctrl        *gomock.Controller
}

func (s *AuthHandlerSuite) SetupSuite() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.mockService = mocks.NewMockAuthService(s.ctrl)
	s.handler = NewAuthHandler(s.mockService)
	mux := http.NewServeMux()
	s.handler.Register(mux)
	s.router = *mux
}
func (s *AuthHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *AuthHandlerSuite) TestService_Authorize() {
	s.T().Run("user is found and authorized", func(t *testing.T) {
		req := &authModel.AuthorizationRequest{
			ClientID:    "test-client-id",
			Scopes:      []string{"scope1", "scope2"},
			RedirectURI: "some-redirect-uri/",
			State:       "test-state",
		}
		expectedResp := &authModel.AuthorizationResult{
			SessionID:   "sess_12345",
			RedirectURI: "some-redirect-uri/",
		}
		s.mockService.EXPECT().Authorize(s.ctx, req).Return(expectedResp, nil)

		body, err := json.Marshal(req)
		require.NoError(t, err)
		httpReq := httptest.NewRequest(http.MethodPost, "/auth/authorize", strings.NewReader(string(body)))
		httpReq.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.router.ServeHTTP(rr, httpReq)

		var got map[string]string
		s.Require().NoError(json.NewDecoder(rr.Body).Decode(&got))
		s.Equal(expectedResp.SessionID, got["session_id"])
		s.Equal(expectedResp.RedirectURI, got["redirect_uri"])
	})
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}
