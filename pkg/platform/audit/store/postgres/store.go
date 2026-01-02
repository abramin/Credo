package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	id "credo/pkg/domain"
	audit "credo/pkg/platform/audit"
	txcontext "credo/pkg/platform/tx"

	"github.com/google/uuid"
)

// Store implements audit.Store using the transactional outbox pattern.
// Events are written to the outbox table and published to Kafka by the outbox worker.
// Kafka is the source of truth for audit events.
type Store struct {
	db *sql.DB
}

// New creates a new PostgreSQL audit store that writes to the outbox.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *Store) execer(ctx context.Context) dbExecutor {
	if tx, ok := txcontext.From(ctx); ok {
		return tx
	}
	return s.db
}

// outboxPayload is the JSON structure published to Kafka.
// Field names match audit.Event for proper deserialization by the consumer.
type outboxPayload struct {
	ID              string `json:"ID"`
	Category        string `json:"Category"`
	Timestamp       string `json:"Timestamp"`
	UserID          string `json:"UserID,omitempty"`
	Subject         string `json:"Subject"`
	Action          string `json:"Action"`
	Purpose         string `json:"Purpose,omitempty"`
	RequestingParty string `json:"RequestingParty,omitempty"`
	Decision        string `json:"Decision,omitempty"`
	Reason          string `json:"Reason,omitempty"`
	Email           string `json:"Email,omitempty"`
	RequestID       string `json:"RequestID,omitempty"`
	ActorID         string `json:"ActorID,omitempty"`
}

// Append writes an audit event to the outbox table for Kafka publishing.
func (s *Store) Append(ctx context.Context, event audit.Event) error {
	eventID := uuid.New()

	// Always derive category from action - eventCategories map is the source of truth
	category := audit.AuditEvent(event.Action).Category()

	// Build JSON payload for Kafka
	payload := outboxPayload{
		ID:              eventID.String(),
		Category:        string(category),
		Timestamp:       event.Timestamp.Format(time.RFC3339Nano),
		Subject:         event.Subject,
		Action:          event.Action,
		Purpose:         event.Purpose,
		RequestingParty: event.RequestingParty,
		Decision:        event.Decision,
		Reason:          event.Reason,
		Email:           event.Email,
		RequestID:       event.RequestID,
		ActorID:         event.ActorID,
	}
	if !event.UserID.IsNil() {
		payload.UserID = uuid.UUID(event.UserID).String()
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}

	// Determine aggregate type and ID
	aggregateType := "audit"
	aggregateID := eventID.String()
	if !event.UserID.IsNil() {
		aggregateType = "user"
		aggregateID = uuid.UUID(event.UserID).String()
	}

	query := `
		INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = s.execer(ctx).ExecContext(ctx, query,
		uuid.New(), // outbox entry ID
		aggregateType,
		aggregateID,
		event.Action,
		payloadBytes,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert outbox entry: %w", err)
	}
	return nil
}

// AppendWithID inserts an audit event into the audit_events table with a specific ID.
// Used by the Kafka consumer to materialize events for querying.
// This is idempotent - duplicate inserts are ignored via ON CONFLICT DO NOTHING.
func (s *Store) AppendWithID(ctx context.Context, eventID uuid.UUID, event audit.Event) error {
	query := `
		INSERT INTO audit_events (
			id, category, timestamp, user_id, subject, action,
			purpose, requesting_party, decision, reason,
			email, request_id, actor_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO NOTHING
	`

	var userID *uuid.UUID
	if !event.UserID.IsNil() {
		uid := uuid.UUID(event.UserID)
		userID = &uid
	}

	_, err := s.db.ExecContext(ctx, query,
		eventID,
		string(event.Category),
		event.Timestamp,
		userID,
		event.Subject,
		event.Action,
		event.Purpose,
		event.RequestingParty,
		event.Decision,
		event.Reason,
		event.Email,
		event.RequestID,
		event.ActorID,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// ListByUser returns events for a specific user.
func (s *Store) ListByUser(ctx context.Context, userID id.UserID) ([]audit.Event, error) {
	query := `
		SELECT category, timestamp, user_id, subject, action,
			   purpose, requesting_party, decision, reason,
			   email, request_id, actor_id
		FROM audit_events
		WHERE user_id = $1
		ORDER BY timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, uuid.UUID(userID))
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	return s.scanEvents(rows)
}

// ListAll returns all audit events (admin only).
func (s *Store) ListAll(ctx context.Context) ([]audit.Event, error) {
	query := `
		SELECT category, timestamp, user_id, subject, action,
			   purpose, requesting_party, decision, reason,
			   email, request_id, actor_id
		FROM audit_events
		ORDER BY timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	return s.scanEvents(rows)
}

// ListRecent returns the N most recent events.
func (s *Store) ListRecent(ctx context.Context, limit int) ([]audit.Event, error) {
	query := `
		SELECT category, timestamp, user_id, subject, action,
			   purpose, requesting_party, decision, reason,
			   email, request_id, actor_id
		FROM audit_events
		ORDER BY timestamp DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	return s.scanEvents(rows)
}

// scanEvents scans multiple rows into audit.Event slice.
func (s *Store) scanEvents(rows *sql.Rows) ([]audit.Event, error) {
	var events []audit.Event

	for rows.Next() {
		var (
			category       string
			event          audit.Event
			userIDNullable *uuid.UUID
		)

		err := rows.Scan(
			&category,
			&event.Timestamp,
			&userIDNullable,
			&event.Subject,
			&event.Action,
			&event.Purpose,
			&event.RequestingParty,
			&event.Decision,
			&event.Reason,
			&event.Email,
			&event.RequestID,
			&event.ActorID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}

		event.Category = audit.EventCategory(category)
		if userIDNullable != nil {
			event.UserID = id.UserID(*userIDNullable)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}

	return events, nil
}

// -----------------------------------------------------------------------------
// Category-specific storage methods for partitioned tables
// -----------------------------------------------------------------------------

// ComplianceRecord represents a compliance audit event for the audit_compliance table.
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

// AppendCompliance inserts a compliance event into the audit_compliance table.
// Idempotent via ON CONFLICT DO NOTHING.
func (s *Store) AppendCompliance(ctx context.Context, eventID uuid.UUID, record ComplianceRecord) error {
	query := `
		INSERT INTO audit_compliance (
			id, timestamp, user_id, subject, action,
			purpose, decision, subject_id_hash, request_id, actor_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		eventID,
		record.Timestamp,
		uuid.UUID(record.UserID),
		record.Subject,
		record.Action,
		record.Purpose,
		record.Decision,
		record.SubjectIDHash,
		record.RequestID,
		record.ActorID,
	)
	if err != nil {
		return fmt.Errorf("insert compliance event: %w", err)
	}
	return nil
}

// SecurityRecord represents a security audit event for the audit_security table.
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

// AppendSecurity inserts a security event into the audit_security table.
// Idempotent via ON CONFLICT DO NOTHING.
func (s *Store) AppendSecurity(ctx context.Context, eventID uuid.UUID, record SecurityRecord) error {
	query := `
		INSERT INTO audit_security (
			id, timestamp, subject, action, reason,
			ip, request_id, actor_id, severity
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		eventID,
		record.Timestamp,
		record.Subject,
		record.Action,
		record.Reason,
		record.IP,
		record.RequestID,
		record.ActorID,
		record.Severity,
	)
	if err != nil {
		return fmt.Errorf("insert security event: %w", err)
	}
	return nil
}

// OpsRecord represents an operational audit event for the audit_ops table.
type OpsRecord struct {
	Timestamp time.Time
	Subject   string
	Action    string
	RequestID string
}

// AppendOps inserts an ops event into the audit_ops table.
// Idempotent via ON CONFLICT DO NOTHING.
func (s *Store) AppendOps(ctx context.Context, eventID uuid.UUID, record OpsRecord) error {
	query := `
		INSERT INTO audit_ops (
			id, timestamp, subject, action, request_id
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id, timestamp) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		eventID,
		record.Timestamp,
		record.Subject,
		record.Action,
		record.RequestID,
	)
	if err != nil {
		return fmt.Errorf("insert ops event: %w", err)
	}
	return nil
}
