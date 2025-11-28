// Package sqlite provides SQLite storage operations for task management in the queue system.
package sqlite

import (
	"github.com/jmoiron/sqlx"
)

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
