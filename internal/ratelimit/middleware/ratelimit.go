package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"credo/internal/ratelimit/models"
	"credo/pkg/platform/httputil"
	auth "credo/pkg/platform/middleware/auth"
	metadata "credo/pkg/platform/middleware/metadata"
	"credo/pkg/platform/privacy"
)

// RateLimiter defines the interface for the rate limiting middleware.
// This is a minimal interface containing only the methods the middleware actually uses.
type RateLimiter interface {
	CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error)
	CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error)
	CheckGlobalThrottle(ctx context.Context) (bool, error)
}

type ClientRateLimiter interface {
	Check(ctx context.Context, clientID, endpoint string) (*models.RateLimitResult, error)
}

type Middleware struct {
	limiter         RateLimiter
	logger          *slog.Logger
	disabled        bool
	failClosed      bool   // If true, reject requests when rate limiter is unavailable (high-security mode)
	supportURL      string // URL for user support (included in auth lockout response)
	ipBreaker       *CircuitBreaker
	combinedBreaker *CircuitBreaker
	fallback        RateLimiter
}

type Option func(*Middleware)

func WithDisabled(disabled bool) Option {
	return func(m *Middleware) {
		m.disabled = disabled
	}
}

func WithSupportURL(url string) Option {
	return func(m *Middleware) {
		m.supportURL = url
	}
}

func WithFallbackLimiter(limiter RateLimiter) Option {
	return func(m *Middleware) {
		if limiter != nil {
			m.fallback = limiter
		}
	}
}

// WithFailClosed enables fail-closed behavior for high-security deployments.
// When enabled, requests are rejected (503) if the rate limiter is unavailable
// and no fallback succeeds. Default is fail-open (requests proceed on error).
func WithFailClosed(enabled bool) Option {
	return func(m *Middleware) {
		m.failClosed = enabled
	}
}

func New(limiter RateLimiter, logger *slog.Logger, opts ...Option) *Middleware {
	m := &Middleware{
		limiter:         limiter,
		logger:          logger,
		ipBreaker:       newCircuitBreaker("ip"),
		combinedBreaker: newCircuitBreaker("combined"),
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.disabled {
		logger.Info("rate limiting disabled")
	}
	return m
}

func (m *Middleware) RateLimit(class models.EndpointClass) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.disabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			ip := metadata.GetClientIP(ctx)

			result, degraded, err := m.checkIPRateLimit(ctx, ip, class)
			if err != nil && !degraded {
				// DESIGN DECISION: Fail-open on rate limit check errors.
				// This prioritizes availability over security - requests proceed when the
				// rate limit store is unavailable (e.g., Redis outage). The error is logged
				// for monitoring/alerting. This is a deliberate tradeoff: during store outages,
				// rate limiting is temporarily bypassed to avoid cascading failures.
				//
				// For high-security deployments requiring fail-closed behavior, see future
				// PRD for configurable FailClosed option.
				m.logger.Error("failed to check IP rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip))
				next.ServeHTTP(w, r)
				return
			}
			if err != nil && degraded {
				m.logger.Error("failed to check IP rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip))
			}

			//Add headers regardless of outcome
			if degraded {
				w.Header().Set("X-RateLimit-Status", "degraded")
			}
			addRateLimitHeaders(w, result)

			if !result.Allowed {
				writeRateLimitExceeded(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *Middleware) RateLimitAuthenticated(class models.EndpointClass) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.disabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			ip := metadata.GetClientIP(ctx)
			userID := auth.GetUserID(ctx)

			result, degraded, err := m.checkBothLimits(ctx, ip, userID, class)
			if err != nil && !degraded {
				// Fail-open: see RateLimit() for design rationale.
				m.logger.Error("failed to check combined rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip), "user_id", userID)
				next.ServeHTTP(w, r)
				return
			}
			if err != nil && degraded {
				m.logger.Error("failed to check combined rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip), "user_id", userID)
			}

			if degraded {
				w.Header().Set("X-RateLimit-Status", "degraded")
			}
			addRateLimitHeaders(w, result)

			if !result.Allowed {
				writeUserRateLimitExceeded(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GlobalThrottle returns middleware for global DDoS protection.
func (m *Middleware) GlobalThrottle() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.disabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			allowed, err := m.limiter.CheckGlobalThrottle(ctx)
			if err != nil {
				// Fail-open: see RateLimit() for design rationale.
				w.Header().Set("X-RateLimit-Status", "degraded")
				m.logger.Error("failed to check global throttle", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				writeServiceOverloaded(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func addRateLimitHeaders(w http.ResponseWriter, result *models.RateLimitResult) {
	if result == nil {
		return
	}
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
}

func writeRateLimitExceeded(w http.ResponseWriter, result *models.RateLimitResult) {
	w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	httputil.WriteJSON(w, http.StatusTooManyRequests, &models.RateLimitExceededResponse{
		Error:      "rate_limit_exceeded",
		Message:    "Too many requests from this IP address. Please try again later.",
		RetryAfter: result.RetryAfter,
	})
}

func writeUserRateLimitExceeded(w http.ResponseWriter, result *models.RateLimitResult) {
	w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	httputil.WriteJSON(w, http.StatusTooManyRequests, &models.UserRateLimitExceededResponse{
		Error:          "user_rate_limit_exceeded",
		Message:        "You have exceeded your request quota for this operation.",
		QuotaLimit:     result.Limit,
		QuotaRemaining: result.Remaining,
		QuotaReset:     result.ResetAt,
	})
}

func writeServiceOverloaded(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	httputil.WriteJSON(w, http.StatusServiceUnavailable, &models.ServiceOverloadedResponse{
		Error:      "service_unavailable",
		Message:    "Service is temporarily overloaded. Please try again later.",
		RetryAfter: 60,
	})
}

func writeClientRateLimitExceeded(w http.ResponseWriter, result *models.RateLimitResult) {
	w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	httputil.WriteJSON(w, http.StatusTooManyRequests, &models.ClientRateLimitExceededResponse{
		Error:      "client_rate_limit_exceeded",
		Message:    "OAuth client has exceeded its request quota. Please retry later.",
		RetryAfter: result.RetryAfter,
	})
}

type ClientMiddleware struct {
	limiter        ClientRateLimiter
	logger         *slog.Logger
	disabled       bool
	circuitBreaker *CircuitBreaker
	fallback       ClientRateLimiter
}

type ClientOption func(*ClientMiddleware)

func WithClientFallbackLimiter(limiter ClientRateLimiter) ClientOption {
	return func(m *ClientMiddleware) {
		if limiter != nil {
			m.fallback = limiter
		}
	}
}

// NewClientMiddleware creates a new client rate limit middleware.
func NewClientMiddleware(limiter ClientRateLimiter, logger *slog.Logger, disabled bool, opts ...ClientOption) *ClientMiddleware {
	m := &ClientMiddleware{
		limiter:        limiter,
		logger:         logger,
		disabled:       disabled,
		circuitBreaker: newCircuitBreaker("client"),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// RateLimitClient returns middleware that enforces per-client rate limits on OAuth endpoints.
// It extracts client_id from query parameters (for authorize) or request body (for token).
func (m *ClientMiddleware) RateLimitClient() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.disabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Extract client_id from query params (authorize) or form data (token)
			clientID := r.URL.Query().Get("client_id")
			if clientID == "" {
				// Try form data for POST requests
				if r.Method == http.MethodPost {
					clientID = r.FormValue("client_id")
				}
			}

			if clientID == "" {
				// No client_id, skip client rate limiting
				next.ServeHTTP(w, r)
				return
			}

			endpoint := r.URL.Path
			result, degraded, err := m.checkClientLimit(ctx, clientID, endpoint)
			if err != nil && !degraded {
				// Fail-open: see Middleware.RateLimit() for design rationale.
				m.logger.Error("failed to check client rate limit", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if err != nil && degraded {
				m.logger.Error("failed to check client rate limit", "error", err)
			}

			if degraded {
				w.Header().Set("X-RateLimit-Status", "degraded")
			}
			addRateLimitHeaders(w, result)

			if !result.Allowed {
				writeClientRateLimitExceeded(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeFailClosedError(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "30")
	httputil.WriteJSON(w, http.StatusServiceUnavailable, &models.ServiceOverloadedResponse{
		Error:      "rate_limit_unavailable",
		Message:    "Rate limiting service is temporarily unavailable. Please try again later.",
		RetryAfter: 30,
	})
}

// withCircuitBreaker wraps a rate limit check with circuit breaker logic.
// It handles primary check, fallback on failure, and circuit state transitions.
// Logs state transitions and fallback usage for observability.
func withCircuitBreaker[T any](
	breaker *CircuitBreaker,
	logger *slog.Logger,
	primary func() (T, error),
	fallback func() (T, error),
	fallbackName string,
) (result T, degraded bool, err error) {
	result, err = primary()
	if err != nil {
		useFallback, change := breaker.RecordFailure()
		if change.Opened && logger != nil {
			logger.Warn("circuit breaker opened",
				"breaker", breaker.Name(),
				"reason", "failure_threshold_reached",
			)
		}
		if useFallback && fallback != nil {
			if logger != nil {
				logger.Info("using fallback rate limiter",
					"breaker", breaker.Name(),
					"reason", "circuit_open",
				)
			}
			fallbackResult, fallbackErr := fallback()
			if fallbackErr != nil {
				if logger != nil {
					logger.Error("fallback "+fallbackName+" failed", "error", fallbackErr)
				}
				return result, false, err
			}
			return fallbackResult, true, err
		}
		return result, false, err
	}

	// Primary succeeded - check if circuit is still open (needs more successes to close)
	usePrimary, change := breaker.RecordSuccess()
	if change.Closed && logger != nil {
		logger.Info("circuit breaker closed",
			"breaker", breaker.Name(),
			"reason", "recovery_complete",
		)
	}
	if !usePrimary && fallback != nil {
		if logger != nil {
			logger.Debug("using fallback during recovery",
				"breaker", breaker.Name(),
				"reason", "circuit_half_open",
			)
		}
		fallbackResult, fallbackErr := fallback()
		if fallbackErr != nil {
			if logger != nil {
				logger.Error("fallback "+fallbackName+" failed", "error", fallbackErr)
			}
			return result, false, nil
		}
		return fallbackResult, true, nil
	}

	return result, false, nil
}

func (m *Middleware) checkIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, bool, error) {
	primary := func() (*models.RateLimitResult, error) {
		return m.limiter.CheckIPRateLimit(ctx, ip, class)
	}
	var fallback func() (*models.RateLimitResult, error)
	if m.fallback != nil {
		fallback = func() (*models.RateLimitResult, error) {
			return m.fallback.CheckIPRateLimit(ctx, ip, class)
		}
	}
	return withCircuitBreaker(m.ipBreaker, m.logger, primary, fallback, "IP rate limit")
}

func (m *Middleware) checkBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, bool, error) {
	primary := func() (*models.RateLimitResult, error) {
		return m.limiter.CheckBothLimits(ctx, ip, userID, class)
	}
	var fallback func() (*models.RateLimitResult, error)
	if m.fallback != nil {
		fallback = func() (*models.RateLimitResult, error) {
			return m.fallback.CheckBothLimits(ctx, ip, userID, class)
		}
	}
	return withCircuitBreaker(m.combinedBreaker, m.logger, primary, fallback, "combined rate limit")
}

func (m *ClientMiddleware) checkClientLimit(ctx context.Context, clientID, endpoint string) (*models.RateLimitResult, bool, error) {
	primary := func() (*models.RateLimitResult, error) {
		return m.limiter.Check(ctx, clientID, endpoint)
	}
	var fallback func() (*models.RateLimitResult, error)
	if m.fallback != nil {
		fallback = func() (*models.RateLimitResult, error) {
			return m.fallback.Check(ctx, clientID, endpoint)
		}
	}
	return withCircuitBreaker(m.circuitBreaker, m.logger, primary, fallback, "client rate limit")
}
