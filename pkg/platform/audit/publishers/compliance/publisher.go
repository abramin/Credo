// Package compliance provides a fail-closed audit publisher for regulatory events.
//
// ComplianceAuditor emits compliance events with synchronous, fail-closed semantics.
// Events are written to the outbox and the caller blocks until the write succeeds.
// If the write fails, an error is returned and the calling operation MUST fail.
//
// Use for: user_created, user_deleted, consent_*, decision_made
package compliance

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	audit "credo/pkg/platform/audit"
)

// Publisher emits compliance events with fail-closed semantics.
// All writes are synchronous - the caller blocks until persistence succeeds or fails.
type Publisher struct {
	store   audit.Store
	logger  *slog.Logger
	metrics *Metrics
}

// Option configures the Publisher.
type Option func(*Publisher)

// WithLogger sets a logger for error reporting.
func WithLogger(logger *slog.Logger) Option {
	return func(p *Publisher) {
		p.logger = logger
	}
}

// WithMetrics sets the metrics collector.
func WithMetrics(m *Metrics) Option {
	return func(p *Publisher) {
		p.metrics = m
	}
}

// New creates a compliance publisher.
// The store must be outbox-backed for guaranteed delivery.
func New(store audit.Store, opts ...Option) *Publisher {
	p := &Publisher{
		store: store,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Emit synchronously writes a compliance event to the audit store.
// Returns error if persistence fails - the caller MUST fail its operation.
//
// This is a fail-closed operation: if the audit cannot be persisted,
// the business operation must not proceed.
func (p *Publisher) Emit(ctx context.Context, event audit.ComplianceEvent) error {
	start := time.Now()

	// Validate required fields
	if event.UserID.IsNil() {
		return fmt.Errorf("compliance event requires UserID")
	}
	if event.Action == "" {
		return fmt.Errorf("compliance event requires Action")
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert to legacy Event for store compatibility
	legacyEvent := event.ToLegacyEvent()

	// Synchronous write - this is the critical path
	if err := p.store.Append(ctx, legacyEvent); err != nil {
		if p.metrics != nil {
			p.metrics.IncPersistFailures()
		}
		if p.logger != nil {
			p.logger.ErrorContext(ctx, "CRITICAL: compliance audit failed",
				"action", event.Action,
				"user_id", event.UserID,
				"error", err,
			)
		}
		return fmt.Errorf("compliance audit persistence failed: %w", err)
	}

	if p.metrics != nil {
		p.metrics.ObservePersistDuration(time.Since(start).Seconds())
		p.metrics.IncEventsEmitted()
	}

	return nil
}

// Close is a no-op for the synchronous compliance publisher.
func (p *Publisher) Close() error {
	return nil
}
