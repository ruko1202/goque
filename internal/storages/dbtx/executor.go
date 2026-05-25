// Package dbtx provides a thin transaction-aware database executor
// for storage backends. It carries an optional *sqlx.Tx through
// context, lets storage code stay unaware of whether it is running
// inside a caller-owned transaction, and exposes the WithinTx helper
// for opening/committing/rolling back a transaction with panic
// recovery.
package dbtx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DBTx is the read/write surface shared by *sqlx.DB and *sqlx.Tx, so
// storage code can hold either without caring which it is.
type DBTx interface {
	Get(dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	Select(dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error

	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DB wraps a *sqlx.DB and picks the right executor at call time —
// the ctx-attached *sqlx.Tx if present, otherwise the underlying
// connection pool.
type DB struct {
	db *sqlx.DB
}

// NewDB returns a DB wrapper over the given *sqlx.DB.
func NewDB(db *sqlx.DB) *DB {
	return &DB{db: db}
}

// GetDB returns the underlying *sqlx.DB. Use it when you need the
// connection pool directly (e.g. to BeginTxx, or for code paths that
// must never enroll in a caller's tx).
func (t *DB) GetDB() *sqlx.DB {
	return t.db
}

// Executor returns the *sqlx.Tx carried by ctx (via WithTx) if any,
// otherwise the underlying *sqlx.DB. Both satisfy DBTx, so callers
// can issue ExecContext/QueryContext on the result uniformly.
func (t *DB) Executor(ctx context.Context) DBTx {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return t.db
}
