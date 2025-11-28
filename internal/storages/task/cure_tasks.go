package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// CureTasks marks stuck tasks in the specified unhealthy status as errored within a transaction.
func (s *Storage) CureTasks(ctx context.Context, unhealthStatus entity.TaskStatus, updatedAtTimeAgo time.Duration, limit int64) ([]*entity.Task, error) {
	var tasks []*entity.Task
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getOlderTasks(ctx, tx, &GetTasksFilter{
			Status:           &unhealthStatus,
			UpdatedAtTimeAgo: &updatedAtTimeAgo,
		}, limit, true)
		if err != nil {
			return err
		}

		for _, task := range tasks {
			task.Status = entity.TaskStatusError
			task.AddError(fmt.Errorf("stuck task"))

			if err := s.updateTask(ctx, tx, task.ID, task); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to cure tasks", slog.Any("err", err))
		return nil, err
	}
	return tasks, nil
}
