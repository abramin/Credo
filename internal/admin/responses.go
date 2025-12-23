package admin

import "time"

// UserInfoResponse is the HTTP response DTO for user info.
type UserInfoResponse struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	SessionCount int       `json:"session_count"`
	LastActive   time.Time `json:"last_active"`
	Verified     bool      `json:"verified"`
}

// UsersListResponse wraps the list of users for HTTP response.
type UsersListResponse struct {
	Users []*UserInfoResponse `json:"users"`
	Total int                 `json:"total"`
}
