// Package observability provides audit logging helpers for the ratelimit module.
package observability

import (
	"context"
	"log/slog"

	"credo/pkg/platform/attrs"
	"credo/pkg/platform/audit"
	"credo/pkg/platform/audit/publishers/security"
	"credo/pkg/requestcontext"
)

// AuditPublisher is a type alias to the security publisher for security-relevant operations.
type AuditPublisher = *security.Publisher

// LogAudit logs audit events to both structured logger and audit publisher.
// It enriches events with request ID and extracts subject/reason from attrList.
func LogAudit(ctx context.Context, logger *slog.Logger, publisher AuditPublisher, event string, attrList ...any) {
	requestID := requestcontext.RequestID(ctx)

	if requestID != "" {
		attrList = append(attrList, "request_id", requestID)
	}

	args := append(attrList, "event", event, "log_type", "audit")

	if logger != nil {
		logger.InfoContext(ctx, event, args...)
	}

	if publisher == nil {
		return
	}

	publisher.Emit(ctx, audit.SecurityEvent{
		Action:    event,
		Subject:   extractSubject(attrList),
		RequestID: requestID,
		Reason:    extractReason(attrList),
		Severity:  audit.SeverityWarning,
	})
}

func extractSubject(attrList []any) string {
	for _, key := range []string{"identifier", "ip", "user_id", "client_id", "api_key_id"} {
		if val := attrs.ExtractString(attrList, key); val != "" {
			return val
		}
	}
	return ""
}

func extractReason(attrList []any) string {
	for _, key := range []string{"reason", "bypass_type"} {
		if val := attrs.ExtractString(attrList, key); val != "" {
			return val
		}
	}
	return ""
}
