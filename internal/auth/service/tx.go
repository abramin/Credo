package service

import (
	"context"
)

// AuthStoreTx provides a transactional boundary for auth-related store mutations.
// Implementations may wrap a database transaction or, in-memory, a coarse lock.
// The txAuthStores parameter provides access to stores within the transaction scope.
type AuthStoreTx interface {
	RunInTx(ctx context.Context, fn func(stores txAuthStores) error) error
}
