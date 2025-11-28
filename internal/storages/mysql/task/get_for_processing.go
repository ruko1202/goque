package mysqltask

import (
	"context"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.GetTasksForProcessing")

	var tasks []*model.Task
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
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

	return fromDBModels(tasks)
}

func (s *Storage) getTasksForProcessingTx(ctx context.Context, tx *sqlx.Tx, taskType entity.TaskType, limit int64) ([]*model.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(
			mysql.AND(
				table.Task.Type.EQ(mysql.String(taskType)),
				table.Task.Status.IN(
					mysql.String(entity.TaskStatusNew),
					mysql.String(entity.TaskStatusError),
				),
				table.Task.NextAttemptAt.LT_EQ(mysql.TimestampT(xtime.Now())),
			),
		).
		FOR(mysql.UPDATE()).
		ORDER_BY(
			table.Task.NextAttemptAt.ASC(),
		).
		LIMIT(limit)

	query, args := stmt.Sql()

	tasks := make([]*model.Task, 0)
	err := tx.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to get task for processing", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}
