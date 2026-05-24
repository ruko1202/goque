package dbtx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
)

// WithinTx runs fn inside a *sqlx.Tx opened on db. The tx is attached
// to ctx via WithTx so any storage call inside fn that uses Executor
// picks it up. On panic or error fn's return is joined with the
// rollback result; on clean return the tx is committed. Errors from
// BeginTxx, Rollback, and Commit are surfaced and logged.
func WithinTx(ctx context.Context, db *sqlx.DB, fn func(ctx context.Context) error) (err error) {
	ctx, span := xlog.WithOperationSpan(ctx, "txManager.WithinTx")
	defer span.End()

	tx, beginTxErr := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if beginTxErr != nil {
		return beginTxErr
	}

	ctx = WithTx(ctx, tx)

	defer func() {
		if recErr := recover(); recErr != nil {
			xlog.Error(ctx, "panic recovery when execute the transaction", xfield.Any("panic", recErr))
			err = errors.Join(err, fmt.Errorf("panic recovery: %v", recErr))
		}

		if err != nil {
			if rollbackErr := rollback(ctx, tx); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
				xlog.Error(ctx, "raise error when rollback the transaction", xfield.Error(err))
			}
		}
	}()

	if fnErr := fn(ctx); fnErr != nil {
		err = errors.Join(err, fnErr)
		xlog.Error(ctx, "raise error when execute the transaction", xfield.Error(err))
		return err
	}

	if commitErr := commit(ctx, tx); commitErr != nil {
		err = errors.Join(err, commitErr)
		xlog.Error(ctx, "raise error when commit the transaction", xfield.Error(err))
		return err
	}

	return err
}

func rollback(ctx context.Context, tx *sqlx.Tx) error {
	if err := tx.Rollback(); err != nil {
		xlog.Error(ctx, "failed to rollback the transaction", xfield.Error(err))
		return err
	}
	return nil
}

func commit(ctx context.Context, tx *sqlx.Tx) error {
	if err := tx.Commit(); err != nil {
		xlog.Error(ctx, "failed to commit the transaction", xfield.Error(err))
		return err
	}
	return nil
}
