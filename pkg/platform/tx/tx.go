package tx

import (
	"context"
	"database/sql"
)

type ctxKey struct{}

var txKey = ctxKey{}

// WithTx stores a SQL transaction in context for downstream store usage.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txKey, tx)
}

// From extracts a SQL transaction from context if present.
func From(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}
