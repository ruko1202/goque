package sqlite

import (
	"context"
	"errors"

	"github.com/mattn/go-sqlite3"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.AddTask",
		xfield.String("db.type", "sqlite"),
		xfield.String("task_id", task.ID.String()),
		xfield.String("task_type", task.Type),
	)
	defer span.End()

	// Validate JSON payload before insertion
	if !dbutils.IsValidJSON(task.Payload) {
		return entity.ErrInvalidPayloadFormat
	}

	dbTask := toDBModel(ctx, task)

	stmt := table.GoqueTask.
		INSERT(table.GoqueTask.AllColumns).
		MODEL(dbTask)

	query, args := stmt.Sql()

	_, err := s.db.ExecContext(ctx, query, args...)
	if err := handleError(err); err != nil {
		xlog.Error(ctx, "failed to add task", xfield.Error(err))
		return err
	}

	return nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	// mattn/go-sqlite3 surfaces constraint violations as a typed Error
	// with ExtendedCode encoding the specific constraint that failed.
	// Match on the code rather than the message text so a future driver
	// upgrade or table-name change can't silently break duplicate detection.
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
		return entity.ErrDuplicateTask
	}

	return err
}
