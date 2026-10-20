// Package mysqltask provides MySQL storage operations for task management in the queue system.
package mysqltask

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/storages"
)

var _ storages.Task = (*Storage)(nil)

// Storage handles database operations for tasks.
type Storage struct {
	db *dbtx.DB
}

// NewStorage creates a new Storage instance with the provided database connection.
func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{db: dbtx.NewDB(db)}
}

// GetDB returns the underlying database connection.
func (s *Storage) GetDB() *sqlx.DB {
	return s.db.GetDB()
}

// withinTx runs fn in a tx of goque's own (fetch, cure, delete, reset) at READ
// COMMITTED. Under InnoDB's default REPEATABLE READ the locking range scans in
// these txs also take next-key locks on the index records around the range,
// which belong to tasks workers are moving to another status at the same time;
// the worker's single-row UPDATE and the tx's own UPDATE then wait for each
// other (Error 1213). READ COMMITTED locks only the rows the scan returns.
func (s *Storage) withinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return dbtx.WithinTxIsolation(ctx, s.db.GetDB(), sql.LevelReadCommitted, fn)
}
