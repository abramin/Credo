package service

import (
	"context"
	"sync"
	"time"

	pkgerrors "credo/pkg/domain-errors"
)

// ConsentStoreTx provides a transactional boundary for consent store mutations.
// Implementations may wrap a database transaction or, in-memory, a coarse lock.
type ConsentStoreTx interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context, store Store) error) error
}

// defaultConsentTxTimeout is the maximum duration for a consent transaction.
const defaultConsentTxTimeout = 5 * time.Second

// inMemoryConsentTx provides simple mutex-based transaction support for in-memory stores.
// Used for tests and demo mode. Production uses PostgresTx with real database transactions.
type inMemoryConsentTx struct {
	mu      sync.Mutex
	store   Store
	timeout time.Duration
}

func newInMemoryConsentTx(store Store) *inMemoryConsentTx {
	return &inMemoryConsentTx{store: store}
}

func (t *inMemoryConsentTx) RunInTx(ctx context.Context, fn func(ctx context.Context, store Store) error) error {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return pkgerrors.Wrap(err, pkgerrors.CodeTimeout, "transaction aborted: context cancelled")
	}

	// Apply timeout if not already set
	timeout := t.timeout
	if timeout == 0 {
		timeout = defaultConsentTxTimeout
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Check again after acquiring lock
	if err := ctx.Err(); err != nil {
		return pkgerrors.Wrap(err, pkgerrors.CodeTimeout, "transaction aborted: context cancelled")
	}

	return fn(ctx, t.store)
}

type txUserKey struct{}

var txUserKeyCtx = txUserKey{}
