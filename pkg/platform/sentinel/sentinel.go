package sentinel

import "errors"

// Sentinel errors for infrastructure facts. Stores and infrastructure layers return
// these (optionally wrapped) so services can translate them into domain errors.
//
// These represent factual states about resources, not validation failures:
// - ErrNotFound: entity does not exist in store
// - ErrExpired: token/session/code has expired
// - ErrAlreadyUsed: resource (auth code, refresh token) already consumed
// - ErrInvalidState: entity in wrong state for requested operation
// - ErrUnavailable: service or resource temporarily unavailable
//
// For validation errors (bad input, missing fields), use pkg/domain-errors directly.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrExpired      = errors.New("expired")
	ErrAlreadyUsed  = errors.New("already used")
	ErrInvalidState = errors.New("invalid state")
	ErrUnavailable  = errors.New("unavailable")
)
