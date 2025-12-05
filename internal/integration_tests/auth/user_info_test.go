package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"id-gateway/internal/auth/models"
	"id-gateway/internal/auth/service"
	"id-gateway/internal/auth/store"
	"id-gateway/internal/platform/middleware"
	httptransport "id-gateway/internal/transport/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenIntegration_HappyPath(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userStore := store.NewInMemoryUserStore()
	sessionStore := store.NewInMemorySessionStore()
	svc := service.NewService(userStore, sessionStore, 15*time.Minute)

	handler := httptransport.NewAuthHandler(svc, logger)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	handler.Register(r)

	session := &models.Session{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		ClientID:       "client-123",
		Code:           "valid-auth-code",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		CodeExpiresAt:  time.Now().Add(10 * time.Minute),
		CreatedAt:      time.Now().Add(-15 * time.Minute),
		RequestedScope: []string{"openid", "profile"},
		Status:         service.StatusActive,
		CodeUsed:       false,
		RedirectURI:    "https://client.app/callback",
	}
	err := sessionStore.Save(ctx, session)
	require.NoError(t, err)

	userInfo := &models.User{
		ID:        session.UserID,
		Email:     "user@example.com",
		FirstName: "Test",
		LastName:  "User",
		Verified:  true,
	}
	err = userStore.Save(ctx, userInfo)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer at_sess_"+session.ID.String())
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)

	var userInfoRes models.UserInfoResult
	err = json.NewDecoder(res.Body).Decode(&userInfoRes)
	require.NoError(t, err)

	assert.Equal(t, userInfo.Email, userInfoRes.Email)
	assert.Equal(t, userInfo.FirstName, userInfoRes.GivenName)
	assert.Equal(t, userInfo.LastName, userInfoRes.FamilyName)
	assert.Equal(t, userInfo.FirstName+" "+userInfo.LastName, userInfoRes.Name)
	assert.True(t, userInfoRes.EmailVerified)
}

func TestUserInfo_ErrorScenarios(t *testing.T) {
	// table tests for various error scenarios
	tests := []struct {
		name           string
		setupSession   *models.Session
		setupUser      *models.User
		authHeader     string // Full Authorization header value or empty
		useSessionID   bool   // If true, construct header from setupSession.ID
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing authorization header - 401",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing or invalid access token",
		},
		{
			name:           "invalid token format - missing Bearer prefix",
			authHeader:     "invalid-token-format",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing or invalid access token",
		},
		{
			name:           "invalid token format - missing at_sess_ prefix",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing or invalid access token",
		},
		{
			name:           "invalid token format - Bearer only",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing or invalid access token",
		},
		{
			name:           "invalid session ID in token - 401",
			authHeader:     "Bearer at_sess_invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid session id format",
		},
		{
			name:           "session not found - 401",
			authHeader:     "Bearer at_sess_" + uuid.New().String(),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "session not found",
		},
		{
			name: "session not active - pending consent",
			setupSession: &models.Session{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ClientID:  "client-123",
				Code:      "some-code",
				ExpiresAt: time.Now().Add(10 * time.Minute),
				CreatedAt: time.Now().Add(-15 * time.Minute),
				Status:    service.StatusPendingConsent,
			},
			useSessionID:   true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "session not active",
		},
		{
			name: "user not found - 401",
			setupSession: &models.Session{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ClientID:  "client-123",
				Code:      "some-code",
				ExpiresAt: time.Now().Add(10 * time.Minute),
				CreatedAt: time.Now().Add(-15 * time.Minute),
				Status:    service.StatusActive,
			},
			useSessionID:   true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			userStore := store.NewInMemoryUserStore()
			sessionStore := store.NewInMemorySessionStore()
			svc := service.NewService(userStore, sessionStore, 15*time.Minute)

			handler := httptransport.NewAuthHandler(svc, logger)
			r := chi.NewRouter()
			r.Use(middleware.RequestID)
			handler.Register(r)

			if tt.setupSession != nil {
				err := sessionStore.Save(ctx, tt.setupSession)
				require.NoError(t, err)
			}
			if tt.setupUser != nil {
				err := userStore.Save(ctx, tt.setupUser)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodGet, "/auth/userinfo", nil)

			// Set Authorization header based on test case
			if tt.useSessionID && tt.setupSession != nil {
				// Use the session ID from setupSession
				req.Header.Set("Authorization", "Bearer at_sess_"+tt.setupSession.ID.String())
			} else if tt.authHeader != "" {
				// Use the explicit authHeader value
				req.Header.Set("Authorization", tt.authHeader)
			}
			// If both are empty/false, no Authorization header is set (missing header case)

			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatus, res.StatusCode, "unexpected status code")

			var errBody map[string]string
			err := json.NewDecoder(res.Body).Decode(&errBody)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedError, errBody["error_description"], "unexpected error message")
		})
	}
}
