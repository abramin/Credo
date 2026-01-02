package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"credo/internal/platform/kafka/consumer"

	"github.com/google/uuid"
)

// SecurityHandler processes security audit events from Kafka.
// Events are written to the audit_security table for SIEM integration.
type SecurityHandler struct {
	store  SecurityStore
	logger *slog.Logger
}

// SecurityStore defines the storage interface for security events.
type SecurityStore interface {
	AppendSecurity(ctx context.Context, eventID uuid.UUID, event SecurityRecord) error
}

// SecurityRecord represents a security audit event for storage.
type SecurityRecord struct {
	Timestamp time.Time
	Subject   string
	Action    string
	Reason    string
	IP        string
	RequestID string
	ActorID   string
	Severity  string
}

// NewSecurityHandler creates a security event handler.
func NewSecurityHandler(store SecurityStore, logger *slog.Logger) *SecurityHandler {
	return &SecurityHandler{
		store:  store,
		logger: logger,
	}
}

// securityPayload matches the JSON structure for security events.
type securityPayload struct {
	Timestamp string `json:"Timestamp"`
	Subject   string `json:"Subject"`
	Action    string `json:"Action"`
	Reason    string `json:"Reason"`
	IP        string `json:"IP"`
	RequestID string `json:"RequestID"`
	ActorID   string `json:"ActorID"`
	Severity  string `json:"Severity"`
}

// Handle processes a security audit event.
func (h *SecurityHandler) Handle(ctx context.Context, msg *consumer.Message) error {
	eventID, err := uuid.Parse(string(msg.Key))
	if err != nil {
		h.logger.Warn("failed to parse security event ID",
			"key", string(msg.Key),
			"error", err,
		)
		return nil
	}

	var payload securityPayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Warn("failed to unmarshal security payload",
			"event_id", eventID,
			"error", err,
		)
		return nil
	}

	record := SecurityRecord{
		Subject:   payload.Subject,
		Action:    payload.Action,
		Reason:    payload.Reason,
		IP:        payload.IP,
		RequestID: payload.RequestID,
		ActorID:   payload.ActorID,
		Severity:  payload.Severity,
	}

	// Default severity if not set
	if record.Severity == "" {
		record.Severity = "info"
	}

	// Parse timestamp
	if payload.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, payload.Timestamp); err == nil {
			record.Timestamp = ts
		} else {
			record.Timestamp = time.Now()
		}
	} else {
		record.Timestamp = time.Now()
	}

	// Store security event
	if err := h.store.AppendSecurity(ctx, eventID, record); err != nil {
		h.logger.Error("failed to store security event",
			"event_id", eventID,
			"action", record.Action,
			"error", err,
		)
		return fmt.Errorf("store security event: %w", err)
	}

	h.logger.Debug("stored security event",
		"event_id", eventID,
		"action", record.Action,
		"severity", record.Severity,
	)

	return nil
}
