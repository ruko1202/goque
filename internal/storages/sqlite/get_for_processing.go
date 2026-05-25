package sqlite

import (
	"context"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetTasksForProcessing",
		xfield.String("db.type", "sqlite"),
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

	// SQLite doesn't support FOR UPDATE SKIP LOCKED
	// In WAL mode, the transaction provides row-level locking automatically
	// The forUpdate parameter is kept for interface compatibility but not used
	stmt := table.GoqueTask.
		SELECT(table.GoqueTask.AllColumns).
		WHERE(
			sqlite.AND(
				table.GoqueTask.Type.EQ(sqlite.String(taskType)),
				table.GoqueTask.Status.IN(
					sqlite.String(entity.TaskStatusNew),
					sqlite.String(entity.TaskStatusError),
				),
				table.GoqueTask.NextAttemptAt.LT_EQ(sqlite.String(timeToString(xtime.Now()))),
			),
		).
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
