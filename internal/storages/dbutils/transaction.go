package dbutils

import (
	"context"
	"database/sql"
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
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

// DoInTransaction executes a function within a database transaction with automatic commit/rollback handling.
func DoInTransaction(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	var txErr error
	defer func() {
		if txErr != nil || recover() != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.ErrorContext(ctx, "failed to rollback transaction", slog.Any("err", rollbackErr))
			}
			return
		}

		if commitErr := tx.Commit(); commitErr != nil {
			slog.ErrorContext(ctx, "failed to commit transaction", slog.Any("err", commitErr))
		}
	}()

	txErr = fn(tx)

	return txErr
}
