// Package sqlite provides SQLite storage operations for task management in the queue system.
package sqlite

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/storages"
)

var _ storages.Task = (*Storage)(nil)

// Storage handles database operations for tasks.
type Storage struct {
	db *sqlx.DB
}

// NewStorage creates a new Storage instance with the provided database connection.
func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}

// GetDB returns the underlying database connection.
func (s *Storage) GetDB(ctx context.Context) *sqlx.DB {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetDB", xfield.String("db.type", "sqlite"))
	defer span.End()

	return s.db
}
