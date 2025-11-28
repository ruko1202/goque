package goque

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	sqlitetask "github.com/ruko1202/goque/internal/storages/sqlite"

	mysqltask "github.com/ruko1202/goque/internal/storages/mysql/task"
	pgtask "github.com/ruko1202/goque/internal/storages/pg/task"
	"github.com/ruko1202/goque/pkg/goquestorage"

	"github.com/ruko1202/goque/internal/storages"
)

// TaskStorage defines the interface for task persistence operations.
type TaskStorage = storages.Task

// NewStorage creates a new task storage instance based on the database driver.
// Supports PostgreSQL and MySQL databases.
func NewStorage(db *sqlx.DB) (TaskStorage, error) {
	switch db.DriverName() {
	case goquestorage.PgDriver:
		return pgtask.NewStorage(db), nil
	case goquestorage.MysqlDriver:
		return mysqltask.NewStorage(db), nil
	case goquestorage.SqliteDriver:
		return sqlitetask.NewStorage(db), nil
	default:
		return nil, fmt.Errorf("unsupported db: %s", db.DriverName())
	}
}
