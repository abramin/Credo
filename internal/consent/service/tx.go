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
	RunInTx(ctx context.Context, fn func(store Store) error) error
}

// shardedConsentTx provides fine-grained locking using sharded mutexes.
// Instead of a single global lock, operations are distributed across N shards
// based on a hash of the user ID, reducing contention under concurrent load.
// Increased from 32 to 128 shards for better distribution under high concurrency.
const numConsentShards = 128

// defaultConsentTxTimeout is the maximum duration for a consent transaction.
const defaultConsentTxTimeout = 5 * time.Second

type shardedConsentTx struct {
	shards  [numConsentShards]sync.Mutex
	store   Store
	timeout time.Duration
}

func (t *shardedConsentTx) RunInTx(ctx context.Context, fn func(store Store) error) error {
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

	shard := t.selectShard(ctx)
	t.shards[shard].Lock()
	defer t.shards[shard].Unlock()

	// Check again after acquiring lock
	if err := ctx.Err(); err != nil {
		return pkgerrors.Wrap(err, pkgerrors.CodeTimeout, "transaction aborted: context cancelled")
	}

	return fn(t.store)
}

// selectShard picks a shard based on user ID from context, or defaults to shard 0.
func (t *shardedConsentTx) selectShard(ctx context.Context) int {
	if userID, ok := ctx.Value(txUserKeyCtx).(string); ok && userID != "" {
		return int(hashConsentString(userID) % numConsentShards)
	}
	return 0
}

// hashConsentString uses FNV-1a for better hash distribution than simple multiply-add.
func hashConsentString(s string) uint32 {
	const (
		fnvOffset = 2166136261
		fnvPrime  = 16777619
	)
	h := uint32(fnvOffset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= fnvPrime
	}
	return h
}

type txUserKey struct{}

var txUserKeyCtx = txUserKey{}
