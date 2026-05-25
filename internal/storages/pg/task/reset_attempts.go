package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// ResetAttempts resets the retry attempts counter for a task and sets its status back to new.
func (s *Storage) ResetAttempts(ctx context.Context, id uuid.UUID) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.ResetAttempts",
		xfield.String("task_id", id.String()),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		task, err := s.getTask(ctx, id)
		if err != nil {
			return err
		}

		task.Attempts = 0
		task.Status = entity.TaskStatusNew
		task.NextAttemptAt = xtime.Now()

		taskErr := lo.FromPtr(task.Errors)
		taskErr += fmt.Sprintf("reset attempts: %s\n", task.NextAttemptAt.Format(time.RFC3339))
		task.Errors = &taskErr

		return s.updateTask(ctx, task.ID, task)
	})
	if err != nil {
		return fmt.Errorf("reset attempts failed: %w", err)
	}

	return nil
}
