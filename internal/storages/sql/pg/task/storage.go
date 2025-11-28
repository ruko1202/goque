// Package task provides storage operations for task management in the queue system.
package task

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
