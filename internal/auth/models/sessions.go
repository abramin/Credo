package models

import "time"

type SessionSummary struct {
	SessionID    string    `json:"session_id"`
	Device       string    `json:"device"`
	IPAddress    string    `json:"ip_address,omitempty"`
	Location     string    `json:"location,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	IsCurrent    bool      `json:"is_current"`
}

type SessionsResult struct {
	Sessions []SessionSummary `json:"sessions"`
}
