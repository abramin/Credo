package httptransport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	consentModel "id-gateway/internal/consent/models"
	"id-gateway/internal/platform/metrics"
	"id-gateway/internal/platform/middleware"
	dErrors "id-gateway/pkg/domain-errors"
)

type ConsentService interface {
	Grant(ctx context.Context, userID string, purposes []consentModel.ConsentPurpose, ttl time.Duration) ([]*consentModel.ConsentRecord, error)
	Revoke(ctx context.Context, userID string, purpose consentModel.ConsentPurpose) error
	Require(ctx context.Context, userID string, purpose consentModel.ConsentPurpose, now time.Time) error
	List(ctx context.Context, userID string) ([]*consentModel.ConsentRecord, error)
}

// ConsentHandler handles consent-related endpoints.
type ConsentHandler struct {
	logger       *slog.Logger
	consent      ConsentService
	metrics      *metrics.Metrics
	consentTTL   time.Duration
	jwtValidator middleware.JWTValidator
}

func (h *ConsentHandler) Register(r chi.Router) {
	consentRouter := chi.NewRouter()
	consentRouter.Use(middleware.Recovery(h.logger))
	consentRouter.Use(middleware.RequestID)
	consentRouter.Use(middleware.Logger(h.logger))
	consentRouter.Use(middleware.Timeout(30 * time.Second))
	consentRouter.Use(middleware.ContentTypeJSON)
	consentRouter.Use(middleware.LatencyMiddleware(h.metrics))
	consentRouter.Use(middleware.RequireAuth(h.jwtValidator, h.logger))
	consentRouter.Post("/auth/consent", h.handleGrantConsent)
	consentRouter.Post("/auth/consent/revoke", h.handleRevokeConsent)
	consentRouter.Get("/auth/consent", h.handleGetConsent)

	r.Mount("/", consentRouter)
}

func NewConsentHandler(
	consent ConsentService,
	logger *slog.Logger,
	metrics *metrics.Metrics,
	jwtValidator middleware.JWTValidator) *ConsentHandler {
	return &ConsentHandler{
		logger:       logger,
		consent:      consent,
		metrics:      metrics,
		jwtValidator: jwtValidator,
	}
}

// handleGrantConsent grants consent for the authenticated user.
func (h *ConsentHandler) handleGrantConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)

	// The middleware has already validated the JWT and set the userID in context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		// This should never happen if RequireAuth middleware is configured correctly
		h.logger.ErrorContext(ctx, "userID missing from context despite auth middleware",
			"request_id", requestID,
		)
		writeError(w, dErrors.New(dErrors.CodeInternal, "authentication context error"))
		return
	}

	var grantReq consentModel.GrantConsentRequest
	// Purposes array must not be empty
	err := json.NewDecoder(r.Body).Decode(&grantReq)
	if err != nil {
		h.logger.WarnContext(ctx, "invalid grant consent request",
			"request_id", requestID,
			"error", err.Error(),
		)
		writeError(w, dErrors.New(dErrors.CodeBadRequest, "invalid request body"))
		return
	}

	if len(grantReq.Purposes) == 0 {
		h.logger.WarnContext(ctx, "empty purposes in grant consent request",
			"request_id", requestID,
		)
		writeError(w, dErrors.New(dErrors.CodeBadRequest, "purposes array must not be empty"))
		return
	}
	// Move validation and grant logic to service
	_, err = h.consent.Grant(ctx, userID, grantReq.Purposes, h.consentTTL)
	if err != nil {
		if dErrors.Is(err, dErrors.CodeBadRequest) {
			h.logger.WarnContext(ctx, "invalid grant consent request",
				"request_id", requestID,
				"error", err.Error(),
			)
			writeError(w, err)
			return
		}
		h.logger.ErrorContext(ctx, "failed to grant consent",
			"request_id", requestID,
			"error", err.Error(),
		)
		writeError(w, dErrors.New(dErrors.CodeInternal, "failed to grant consent"))
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func (h *ConsentHandler) handleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/consent/revoke")
}
func (h *ConsentHandler) handleGetConsent(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/consent")
}
func (h *ConsentHandler) notImplemented(w http.ResponseWriter, path string) {
	h.logger.Warn("Not implemented", slog.String("path", path))
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
