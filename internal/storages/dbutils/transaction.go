// Package dbutils provides common database utilities for task storage implementations.
package dbutils

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// DBTx defines the interface for database transaction operations.
type DBTx interface {
	Get(dest interface{}, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	Select(dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error

	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DoInTransaction executes a function within a database transaction with automatic commit/rollback handling.
func DoInTransaction(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	var txErr error
	defer func() {
		if recErr := recover(); recErr != nil {
			slog.ErrorContext(ctx, "panic recovery when execute the transaction", slog.Any("panic", recErr))
			txErr = errors.Join(txErr, rollback(ctx, tx))
			return
		}
		if txErr != nil {
			slog.ErrorContext(ctx, "raise error when execute the transaction", slog.Any("err", txErr))
			txErr = errors.Join(txErr, rollback(ctx, tx))
			return
		}

		txErr = errors.Join(txErr, commit(ctx, tx))
	}()

	txErr = fn(tx)

	return txErr
}

func rollback(ctx context.Context, tx *sqlx.Tx) error {
	if err := tx.Rollback(); err != nil {
		slog.ErrorContext(ctx, "failed to rollback the transaction", slog.Any("err", err))
		return err
	}
	return nil
}

func commit(ctx context.Context, tx *sqlx.Tx) error {
	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "failed to commit the transaction", slog.Any("err", err))
		return err
	}
	return nil
}
