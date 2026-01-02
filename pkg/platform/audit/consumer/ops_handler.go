package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"credo/internal/platform/kafka/consumer"

	"github.com/google/uuid"
)

// OpsHandler processes operational audit events from Kafka.
// Events are written to the audit_ops table with short retention.
type OpsHandler struct {
	store  OpsStore
	logger *slog.Logger
}

// OpsStore defines the storage interface for ops events.
type OpsStore interface {
	AppendOps(ctx context.Context, eventID uuid.UUID, event OpsRecord) error
}

// OpsRecord represents an operational audit event for storage.
type OpsRecord struct {
	Timestamp time.Time
	Subject   string
	Action    string
	RequestID string
}

// NewOpsHandler creates an ops event handler.
func NewOpsHandler(store OpsStore, logger *slog.Logger) *OpsHandler {
	return &OpsHandler{
		store:  store,
		logger: logger,
	}
}

// opsPayload matches the JSON structure for ops events.
type opsPayload struct {
	Timestamp string `json:"Timestamp"`
	Subject   string `json:"Subject"`
	Action    string `json:"Action"`
	RequestID string `json:"RequestID"`
}

// Handle processes an operational audit event.
func (h *OpsHandler) Handle(ctx context.Context, msg *consumer.Message) error {
	eventID, err := uuid.Parse(string(msg.Key))
	if err != nil {
		// Ops events are best-effort - log and continue
		h.logger.Debug("failed to parse ops event ID",
			"key", string(msg.Key),
			"error", err,
		)
		return nil
	}

	var payload opsPayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		h.logger.Debug("failed to unmarshal ops payload",
			"event_id", eventID,
			"error", err,
		)
		return nil
	}

	record := OpsRecord{
		Subject:   payload.Subject,
		Action:    payload.Action,
		RequestID: payload.RequestID,
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

	// Store ops event - errors are logged but don't prevent commit
	if err := h.store.AppendOps(ctx, eventID, record); err != nil {
		h.logger.Debug("failed to store ops event",
			"event_id", eventID,
			"action", record.Action,
			"error", err,
		)
		// Return nil to commit - ops events are best-effort
		return nil
	}

	return nil
}
