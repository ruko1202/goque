package task

import (
	"context"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.GetTasksForProcessing",
		zap.String("task_type", taskType),
	)

	var tasks []*model.Task
	err := dbutils.DoInTransaction(ctx, s.db, func(tx dbutils.DBTx) error {
		var err error
		tasks, err = s.getTasksForProcessingTx(ctx, tx, taskType, limit)
		if err != nil {
			return err
		}

		return s.batchUpdateTasksStatusTx(ctx, tx, tasks, entity.TaskStatusPending)
	})
	if err != nil {
		xlog.Error(ctx, "failed to get task for processing", zap.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks), nil
}

func (s *Storage) getTasksForProcessingTx(ctx context.Context, tx dbutils.DBTx, taskType entity.TaskType, limit int64) ([]*model.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(
			postgres.AND(
				table.Task.Type.EQ(postgres.String(taskType)),
				table.Task.Status.IN(
					postgres.String(entity.TaskStatusNew),
					postgres.String(entity.TaskStatusError),
				),
				table.Task.NextAttemptAt.LT_EQ(postgres.TimestampzT(xtime.Now())),
			),
		).
		FOR(postgres.UPDATE()).
		ORDER_BY(
			table.Task.NextAttemptAt.ASC(),
		).
		LIMIT(limit)

	dbTasks := make([]*model.Task, 0)
	err := stmt.QueryContext(ctx, tx, &dbTasks)
	if err != nil {
		xlog.Error(ctx, "failed to get task for processing", zap.Error(err))
		return nil, err
	}

	return dbTasks, nil
}
