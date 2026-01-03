package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"credo/internal/decision"
	"credo/internal/decision/metrics"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/httputil"
	"credo/pkg/requestcontext"
)

// Handler wires decision endpoints to the decision service.
type Handler struct {
	service *decision.Service
	logger  *slog.Logger
	metrics *metrics.Metrics
}

// New constructs a decision handler with its dependencies.
func New(service *decision.Service, logger *slog.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
		metrics: metrics,
	}
}

// Register mounts decision endpoints on the router.
func (h *Handler) Register(r chi.Router) {
	r.Post("/decision/evaluate", h.HandleEvaluate)
}

// HandleEvaluate handles POST /decision/evaluate requests.
//
// Side effects: validates input, enforces authentication from context, calls the
// decision service, writes an HTTP response, and logs request outcomes.
func (h *Handler) HandleEvaluate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := requestcontext.RequestID(ctx)
	start := time.Now()

	// Require authenticated user
	userID := requestcontext.UserID(ctx)
	if userID == (id.UserID{}) {
		httputil.WriteError(w, dErrors.New(dErrors.CodeUnauthorized, "authentication required"))
		return
	}

	// Decode and validate request
	req, ok := httputil.DecodeAndPrepare[EvaluateRequest](w, r, h.logger, ctx, requestID)
	if !ok {
		return
	}

	// Build domain request
	domainReq := decision.EvaluateRequest{
		UserID:     userID,
		Purpose:    req.ParsedPurpose(),
		NationalID: req.ParsedNationalID(),
	}

	// Call service
	result, err := h.service.Evaluate(ctx, domainReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "decision evaluation failed",
			"request_id", requestID,
			"user_id", userID,
			"purpose", req.Purpose,
			"error", err,
		)
		httputil.WriteError(w, err)
		return
	}

	// Log success
	h.logger.InfoContext(ctx, "decision evaluated",
		"request_id", requestID,
		"user_id", userID,
		"purpose", req.Purpose,
		"status", result.Status,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	// Return response
	httputil.WriteJSON(w, http.StatusOK, FromResult(result))
}
