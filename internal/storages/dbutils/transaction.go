// Package dbutils provides common database utilities for task storage implementations.
package dbutils

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
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
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}

	var txErr error
	defer func() {
		if recErr := recover(); recErr != nil {
			xlog.Error(ctx, "panic recovery when execute the transaction", zap.Any("panic", recErr))
			txErr = errors.Join(txErr, rollback(ctx, tx))
			return
		}
		if txErr != nil {
			xlog.Error(ctx, "raise error when execute the transaction", zap.Error(txErr))
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
		xlog.Error(ctx, "failed to rollback the transaction", zap.Error(err))
		return err
	}
	return nil
}

func commit(ctx context.Context, tx *sqlx.Tx) error {
	if err := tx.Commit(); err != nil {
		xlog.Error(ctx, "failed to commit the transaction", zap.Error(err))
		return err
	}
	return nil
}
