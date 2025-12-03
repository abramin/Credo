package domain

import "time"

// AuditEvent is emitted from domain logic to capture key actions. Keep it
// transport-agnostic so stores and sinks can fan out.
type AuditEvent struct {
	Timestamp       time.Time
	UserID          string // PII
	Subject         string // PII or pseudonymous subject
	Action          string
	Purpose         string
	RequestingParty string
	Decision        string
	Reason          string
}
