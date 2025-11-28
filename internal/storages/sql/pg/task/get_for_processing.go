package task

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	sqldbutils "github.com/ruko1202/goque/internal/storages/sql/utils"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	var tasks []*entity.Task
	err := sqldbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getTasksForProcessingTx(ctx, tx, taskType, limit)
		if err != nil {
			return err
		}

		taskIDs := lo.Map(tasks, func(task *entity.Task, _ int) uuid.UUID {
			task.Status = entity.TaskStatusPending
			return task.ID
		})
		return s.batchUpdateTasksStatusTx(ctx, tx, taskIDs, entity.TaskStatusPending)
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task for processing", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}

func (s *Storage) getTasksForProcessingTx(ctx context.Context, tx *sqlx.Tx, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(
			postgres.AND(
				table.Task.Type.EQ(postgres.String(taskType)),
				table.Task.Status.IN(
					postgres.String(entity.TaskStatusNew),
					postgres.String(entity.TaskStatusError),
				),
				table.Task.NextAttemptAt.LT_EQ(postgres.NOW()),
			),
		).
		ORDER_BY(
			table.Task.NextAttemptAt.ASC(),
		).
		LIMIT(limit)

	tasks, err := s.getTasksTx(ctx, tx, stmt, true)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task for processing", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
