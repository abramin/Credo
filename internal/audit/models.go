package audit

import "time"

// Event is emitted from domain logic to capture key actions. Keep it
// transport-agnostic so stores and sinks can fan out.
type Event struct {
	Timestamp       time.Time
	UserID          string
	Subject         string
	Action          string
	Purpose         string
	RequestingParty string
	Decision        string
	Reason          string
}
