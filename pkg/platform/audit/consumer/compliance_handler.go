package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"credo/internal/platform/kafka/consumer"
	id "credo/pkg/domain"

	"github.com/google/uuid"
)

// ComplianceHandler processes compliance audit events from Kafka.
// Events are written to the audit_compliance table for long-term retention.
type ComplianceHandler struct {
	store  ComplianceStore
	logger *slog.Logger
}

// ComplianceStore defines the storage interface for compliance events.
type ComplianceStore interface {
	AppendCompliance(ctx context.Context, eventID uuid.UUID, event ComplianceRecord) error
}

// ComplianceRecord represents a compliance audit event for storage.
type ComplianceRecord struct {
	Timestamp     time.Time
	UserID        id.UserID
	Subject       string
	Action        string
	Purpose       string
	Decision      string
	SubjectIDHash string
	RequestID     string
	ActorID       string
}

// NewComplianceHandler creates a compliance event handler.
func NewComplianceHandler(store ComplianceStore, logger *slog.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		store:  store,
		logger: logger,
	}
}

// compliancePayload matches the JSON structure for compliance events.
type compliancePayload struct {
	Timestamp     string `json:"Timestamp"`
	UserID        string `json:"UserID"`
	Subject       string `json:"Subject"`
	Action        string `json:"Action"`
	Purpose       string `json:"Purpose"`
	Decision      string `json:"Decision"`
	SubjectIDHash string `json:"SubjectIDHash"`
	RequestID     string `json:"RequestID"`
	ActorID       string `json:"ActorID"`
}

// Handle processes a compliance audit event.
func (h *ComplianceHandler) Handle(ctx context.Context, msg *consumer.Message) error {
	eventID, err := uuid.Parse(string(msg.Key))
	if err != nil {
		h.logger.Error("CRITICAL: failed to parse compliance event ID",
			"key", string(msg.Key),
			"error", err,
		)
		// Return nil to commit - malformed messages should not block
		return nil
	}

	var payload compliancePayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Error("CRITICAL: failed to unmarshal compliance payload",
			"event_id", eventID,
			"error", err,
		)
		return nil
	}

	// Strict validation for compliance events
	if payload.UserID == "" {
		h.logger.Error("CRITICAL: compliance event missing UserID",
			"event_id", eventID,
			"action", payload.Action,
		)
		return nil
	}

	record := ComplianceRecord{
		Subject:       payload.Subject,
		Action:        payload.Action,
		Purpose:       payload.Purpose,
		Decision:      payload.Decision,
		SubjectIDHash: payload.SubjectIDHash,
		RequestID:     payload.RequestID,
		ActorID:       payload.ActorID,
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

	// Parse UserID
	if uid, err := uuid.Parse(payload.UserID); err == nil {
		record.UserID = id.UserID(uid)
	}

	// Store compliance event
	if err := h.store.AppendCompliance(ctx, eventID, record); err != nil {
		h.logger.Error("failed to store compliance event",
			"event_id", eventID,
			"action", record.Action,
			"error", err,
		)
		return fmt.Errorf("store compliance event: %w", err)
	}

	h.logger.Debug("stored compliance event",
		"event_id", eventID,
		"action", record.Action,
		"user_id", record.UserID,
	)

	return nil
}
