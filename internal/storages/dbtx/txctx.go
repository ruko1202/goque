package dbtx

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type txCtxKey struct{}

// WithTx attaches an existing *sqlx.Tx to ctx so that storage operations
// pick it up instead of the storage's own *sqlx.DB. Use this to implement
// the transactional outbox pattern: open a tx, write domain rows, enqueue
// a task via goque, then commit — all atomically.
//
// Subsequent WithTx calls on the same ctx shadow earlier ones (last-wins).
// Pass nil to clear an attached tx (see WithoutTx).
func WithTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// TxFromContext returns the *sqlx.Tx attached via WithTx and true if
// one is present and non-nil; otherwise returns (nil, false). A ctx
// that went through WithoutTx (or never had a tx attached) returns
// (nil, false).
func TxFromContext(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(*sqlx.Tx)

	return tx, ok && tx != nil
}

// WithoutTx returns a context with any attached *sqlx.Tx removed. Use it
// when handing ctx to a code path that must not enroll in the caller's tx
// (e.g. a goroutine that outlives the caller's Commit/Rollback).
func WithoutTx(ctx context.Context) context.Context {
	_, ok := TxFromContext(ctx)
	if ok {
		return context.WithValue(ctx, txCtxKey{}, (*sqlx.Tx)(nil))
	}
	return ctx
}
