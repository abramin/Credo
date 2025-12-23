package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"credo/internal/ratelimit/models"
	"credo/pkg/platform/httputil"
	auth "credo/pkg/platform/middleware/auth"
	metadata "credo/pkg/platform/middleware/metadata"
	"credo/pkg/platform/privacy"
)

type RateLimiter interface {
	CheckIPRateLimit(ctx context.Context, ip string, class models.EndpointClass) (*models.RateLimitResult, error)
	CheckUserRateLimit(ctx context.Context, userID string, class models.EndpointClass) (*models.RateLimitResult, error)
	CheckBothLimits(ctx context.Context, ip, userID string, class models.EndpointClass) (*models.RateLimitResult, error)
	CheckAuthRateLimit(ctx context.Context, identifier, ip string) (*models.AuthRateLimitResult, error)
	CheckGlobalThrottle(ctx context.Context) (bool, error)
}

type Middleware struct {
	limiter  RateLimiter
	logger   *slog.Logger
	disabled bool
}

type Option func(*Middleware)

// WithDisabled disables rate limiting entirely (for testing/demo mode).
func WithDisabled(disabled bool) Option {
	return func(m *Middleware) {
		m.disabled = disabled
	}
}

func New(limiter RateLimiter, logger *slog.Logger, opts ...Option) *Middleware {
	m := &Middleware{
		limiter: limiter,
		logger:  logger,
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

			result, err := m.limiter.CheckIPRateLimit(ctx, ip, class)
			if err != nil {
				m.logger.Error("failed to check IP rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip))
				next.ServeHTTP(w, r)
				return
			}

			//Add headers regardless of outcome
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

			result, err := m.limiter.CheckBothLimits(ctx, ip, userID, class)
			if err != nil {
				m.logger.Error("failed to check combined rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip), "user_id", userID)
				next.ServeHTTP(w, r)
				return
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

// RateLimitAuth returns middleware for authentication endpoints with lockout.
//
// This is applied to /auth/authorize, /auth/token, /auth/password-reset, /mfa/*

func (m *Middleware) RateLimitAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ip := metadata.GetClientIP(ctx)
			identifier := ""
			if r.Method == http.MethodPost && r.Body != nil && r.ContentLength != 0 {
				bodyBytes, err := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				if err == nil && len(bodyBytes) > 0 {
					var payload struct {
						Email    string `json:"email"`
						Username string `json:"username"`
					}
					if jsonErr := json.Unmarshal(bodyBytes, &payload); jsonErr == nil {
						identifier = strings.TrimSpace(payload.Email)
						if identifier == "" {
							identifier = strings.TrimSpace(payload.Username)
						}
					}
				}
			}

			result, err := m.limiter.CheckAuthRateLimit(ctx, identifier, ip)
			if err != nil {
				m.logger.Error("failed to check auth rate limit", "error", err, "ip_prefix", privacy.AnonymizeIP(ip))
				next.ServeHTTP(w, r)
				return
			}

			addRateLimitHeaders(w, &result.RateLimitResult)

			if !result.Allowed {
				writeAuthLockout(w, result)
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

func writeAuthLockout(w http.ResponseWriter, result *models.AuthRateLimitResult) {
	if result == nil {
		return
	}
	w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	httputil.WriteJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":            "rate_limit_exceeded",
		"message":          "Too many authentication attempts. Please try again later.",
		"retry_after":      result.RetryAfter,
		"requires_captcha": result.RequiresCaptcha,
		"failure_count":    result.FailureCount,
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

// Ensure unused imports are referenced
var _ = time.Now
