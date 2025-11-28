package sqlite

import (
	"context"
	"fmt"
	"time"

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
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks updates stuck tasks to error status for retry.
func (s *Storage) CureTasks(
	ctx context.Context,
	taskType entity.TaskType,
	unhealthStatuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.CureTasks")

	tasks := make([]*model.Task, 0)
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getTasksByFilterTx(ctx, tx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         unhealthStatuses,
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			xlog.Error(ctx, "failed to select tasks for deletion", zap.Error(err))
			return err
		}

		return s.cureTaskTx(ctx, tx, tasks, comment)
	})
	if err != nil {
		xlog.Error(ctx, "failed to cure tasks", zap.Error(err))
		return nil, err
	}

	return fromDBModels(tasks)
}

func (s *Storage) cureTaskTx(ctx context.Context, tx dbutils.DBTx, tasks []*model.Task, comment string) error {
	if len(tasks) == 0 {
		return nil
	}

	// SQLite uses || for concatenation and CAST(x AS TEXT) for type conversion
	updateStmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.Errors,
			table.Task.UpdatedAt,
		).
		SET(
			sqlite.String(entity.TaskStatusError),
			// In SQLite: '' || COALESCE(errors, '') || 'attempt ' || <task.Attempts> || ': <comment>\n'
			sqlite.String("").
				CONCAT(sqlite.COALESCE(table.Task.Errors, sqlite.String(""))).
				//comment by format: `attempt <task.Attempts>: <comment>\n`
				CONCAT(sqlite.String("attempt ")).
				CONCAT(table.Task.Attempts).
				CONCAT(sqlite.String(fmt.Sprintf(": %s\n", comment))),
			sqlite.String(timeToString(xtime.Now())),
		).
		WHERE(
			table.Task.ID.IN(lo.Map(tasks, func(item *model.Task, _ int) sqlite.Expression {
				return sqlite.String(lo.FromPtr(item.ID))
			})...),
		)

	query, args := updateStmt.Sql()
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", zap.Error(err))
		return err
	}

	return nil
}
