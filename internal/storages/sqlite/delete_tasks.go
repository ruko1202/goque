package sqlite

import (
	"context"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// DeleteTasks removes tasks with specified statuses that haven't been updated within the given time period.
func (s *Storage) DeleteTasks(
	ctx context.Context,
	taskType entity.TaskType,
	statuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.DeleteTasks",
		zap.Any("statuses", statuses),
		zap.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)

	tasks := make([]*model.Task, 0)
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getTasksByFilterTx(ctx, tx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         statuses,
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			xlog.Error(ctx, "failed to select tasks for deletion", zap.Error(err))
			return err
		}

		return s.deleteTasksTx(ctx, tx, tasks)
	})
	if err != nil {
		xlog.Error(ctx, "failed to delete tasks", zap.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks)
}

func (s *Storage) deleteTasksTx(ctx context.Context, tx dbutils.DBTx, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	stmt := table.Task.DELETE().
		WHERE(
			table.Task.ID.IN(lo.Map(tasks, func(task *model.Task, _ int) mysql.Expression {
				return sqlite.String(lo.FromPtr(task.ID))
			})...),
		)

	query, args := stmt.Sql()
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to delete tasks", zap.Error(err))
		return err
	}

	return nil
}
