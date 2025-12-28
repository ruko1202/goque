package mysqltask

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// ResetAttempts resets the retry attempts counter for a task and sets its status back to new.
func (s *Storage) ResetAttempts(ctx context.Context, id uuid.UUID) error {
	ctx = xlog.WithOperation(ctx, "storage.ResetAttempts",
		zap.String("task_id", id.String()),
	)

	err := dbutils.DoInTransaction(ctx, s.db, func(tx dbutils.DBTx) error {
		task, err := s.getTaskTx(ctx, tx, id)
		if err != nil {
			return err
		}

		task.Attempts = 0
		task.Status = entity.TaskStatusNew
		task.NextAttemptAt = xtime.Now()

		taskErr := lo.FromPtr(task.Errors)
		taskErr += fmt.Sprintf("reset attempts: %s\n", task.NextAttemptAt)
		task.Errors = &taskErr

		return s.updateTaskTx(ctx, tx, task.ID, task)
	})
	if err != nil {
		return fmt.Errorf("reset attempts failed: %w", err)
	}

	return nil
}
