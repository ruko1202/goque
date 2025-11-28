package sqlite

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/storages/dbutils"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	// Validate JSON payload before insertion
	if !dbutils.IsValidJSON(task.Payload) {
		return dbentity.ErrInvalidPayloadFormat
	}

	dbTask := toDBModel(task)

	stmt := table.Task.
		INSERT(table.Task.AllColumns).
		MODEL(dbTask)

	query, args := stmt.Sql()

	_, err := s.db.ExecContext(ctx, query, args...)
	if err := handleError(err); err != nil {
		slog.ErrorContext(ctx, "failed to add task", slog.Any("err", err))
		return err
	}

	return nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	// modernc.org/sqlite returns errors as strings
	// Check for UNIQUE constraint violation
	errMsg := err.Error()
	if errMsg == "UNIQUE constraint failed: task.type, task.external_id" {
		return dbentity.ErrDuplicateTask
	}

	return err
}
