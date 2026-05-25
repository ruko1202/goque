package mysqltask

import (
	"context"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetTasksForProcessing",
		xfield.String("db.type", "mysql"),
		xfield.String("task_type", taskType),
	)
	defer span.End()

	var tasks []*model.GoqueTask
	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		var err error
		tasks, err = s.getTasksForProcessing(ctx, taskType, limit)
		if err != nil {
			return err
		}

		return s.batchUpdateTasksStatus(ctx, tasks, entity.TaskStatusPending)
	})
	if err != nil {
		xlog.Error(ctx, "failed to get task for processing", xfield.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks)
}

func (s *Storage) getTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*model.GoqueTask, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.getTasksForProcessing")
	defer span.End()

	stmt := table.GoqueTask.
		SELECT(table.GoqueTask.AllColumns).
		WHERE(
			mysql.AND(
				table.GoqueTask.Type.EQ(mysql.String(taskType)),
				table.GoqueTask.Status.IN(
					mysql.String(entity.TaskStatusNew),
					mysql.String(entity.TaskStatusError),
				),
				table.GoqueTask.NextAttemptAt.LT_EQ(mysql.TimestampT(xtime.Now())),
			),
		).
		FOR(mysql.UPDATE()).
		ORDER_BY(
			table.GoqueTask.NextAttemptAt.ASC(),
		).
		LIMIT(limit)

	query, args := stmt.Sql()

	tasks := make([]*model.GoqueTask, 0)
	err := s.db.Executor(ctx).SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to get task for processing", xfield.Error(err))
		return nil, err
	}

	return tasks, nil
}
