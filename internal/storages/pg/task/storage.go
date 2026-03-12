// Package task provides storage operations for task management in the queue system.
package task

import (
	"github.com/jmoiron/sqlx"

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
func (s *Storage) GetDB() *sqlx.DB {
	return s.db
}
