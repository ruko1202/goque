package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	ctx = xlog.WithOperation(ctx, "storage.AddTask",
		zap.String("task_id", task.ID.String()),
		zap.String("task_type", task.Type),
	)

	// Validate JSON payload before insertion
	if !dbutils.IsValidJSON(task.Payload) {
		return entity.ErrInvalidPayloadFormat
	}
	dbTask := toDBModel(ctx, task)

	stmt := table.Task.
		INSERT(table.Task.AllColumns).
		MODEL(dbTask)

	_, err := stmt.ExecContext(ctx, s.db)
	if err := handleError(err); err != nil {
		xlog.Error(ctx, "failed to add task", zap.Error(err))
		return err
	}

	return nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code.Name() {
		case "unique_violation":
			return fmt.Errorf("%w: %s", entity.ErrDuplicateTask, pqErr.Detail)
		case "invalid_text_representation":
			return fmt.Errorf("%w: %s", entity.ErrInvalidPayloadFormat, pqErr.Detail)
		default:
			return pqErr
		}
	}

	return err
}
