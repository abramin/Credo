package storage

import pkgerrors "id-gateway/pkg/errors"

var (
	// ErrNotFound keeps storage-specific 404s consistent across in-memory and
	// future implementations.
	ErrNotFound = pkgerrors.New(pkgerrors.CodeNotFound, "record not found")
)
