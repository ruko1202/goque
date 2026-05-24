package mysqltask

import (
	"context"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

// DeleteTasks removes tasks with specified statuses that haven't been updated within the given time period.
func (s *Storage) DeleteTasks(
	ctx context.Context,
	taskType entity.TaskType,
	statuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.DeleteTasks",
		xfield.String("db.type", "mysql"),
		xfield.Any("statuses", statuses),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	defer span.End()

	tasks := make([]*model.GoqueTask, 0)
	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		var err error
		tasks, err = s.getTasksByFilter(ctx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         statuses,
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			xlog.Error(ctx, "failed to select tasks for deletion", xfield.Error(err))
			return err
		}

		return s.deleteTasks(ctx, tasks)
	})
	if err != nil {
		xlog.Error(ctx, "failed to delete tasks", xfield.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks)
}

func (s *Storage) deleteTasks(ctx context.Context, tasks []*model.GoqueTask) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.deleteTasks")
	defer span.End()

	if len(tasks) == 0 {
		return nil
	}
	stmt := table.GoqueTask.DELETE().
		WHERE(
			table.GoqueTask.ID.IN(lo.Map(tasks, func(task *model.GoqueTask, _ int) mysql.Expression {
				return mysql.String(task.ID)
			})...),
		)

	query, args := stmt.Sql()
	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
