package service

//go:generate mockgen -source=service.go -destination=mocks/mocks.go -package=mocks UserStore,SessionStoreimport
import (
	"context"
	"id-gateway/internal/auth/models"
	"id-gateway/internal/auth/service/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ServiceSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	mockUserStore *mocks.MockUserStore
	mockSessStore *mocks.MockSessionStore
	service       *Service
}

func (s *ServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockUserStore = mocks.NewMockUserStore(s.ctrl)
	s.mockSessStore = mocks.NewMockSessionStore(s.ctrl)
	s.service = NewService(s.mockUserStore, s.mockSessStore, 15*time.Minute)
}

func (s *ServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *ServiceSuite) TestAuthorizeHappyPath() {
	s.T().Skip("TODO: implement happy path once Authorize is completed")
	req := models.AuthorizationRequest{
		ClientID:    "client-123",
		Scopes:      []string{"openid", "profile"},
		RedirectURI: "https://client.app/callback",
		State:       "xyz",
		Email:       "email@test.com",
	}
	result, err := s.service.Authorize(context.Background(), &req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "todo-session-id", result.SessionID)
	assert.Equal(s.T(), "https://client.app/callback", result.RedirectURI)
}

func (s *ServiceSuite) TestAuthorizeUserNotFound() {
	s.T().Skip("TODO: implement user not found flow once Authorize is completed")
}

func (s *ServiceSuite) TestAuthorizeUserFound() {
	s.T().Skip("TODO: implement user found flow once Authorize is completed")
}

func (s *ServiceSuite) TestAuthorizeUserStoreError() {
	s.T().Skip("TODO: implement user store error handling once Authorize is completed")
}

func (s *ServiceSuite) TestAuthorizeSessionStoreError() {
	s.T().Skip("TODO: implement session store error handling once Authorize is completed")
}

func (s *ServiceSuite) TestAuthorizeWithState() {
	s.T().Skip("TODO: implement state echo verification once Authorize is completed")
}

func (s *ServiceSuite) TestAuthorizeWithoutState() {
	s.T().Skip("TODO: implement no state handling once Authorize is completed")
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}
