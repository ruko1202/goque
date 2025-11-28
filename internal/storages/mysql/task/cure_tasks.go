package mysqltask

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/utils/xtime"

	"github.com/ruko1202/goque/internal/storages/dbentity"

	"github.com/ruko1202/goque/internal/storages/dbutils"

	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
)

// CureTasks updates stuck tasks to error status for retry.
func (s *Storage) CureTasks(ctx context.Context, taskType entity.TaskType, unhealthStatuses []entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error) {
	tasks := make([]*model.Task, 0)
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getTasksByFilterTx(ctx, tx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         unhealthStatuses,
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			slog.ErrorContext(ctx, "failed to select tasks for deletion", slog.Any("err", err))
			return err
		}

		return s.cureTaskTx(ctx, tx, tasks, comment)
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to cure tasks", slog.Any("err", err))
		return nil, err
	}

	return fromDBModels(tasks)
}

func (s *Storage) cureTaskTx(ctx context.Context, tx dbutils.DBTx, tasks []*model.Task, comment string) error {
	if len(tasks) == 0 {
		return nil
	}
	updateStmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.Errors,
			table.Task.UpdatedAt,
		).
		SET(
			mysql.String(entity.TaskStatusError),
			mysql.CONCAT(
				mysql.COALESCE(table.Task.Errors, mysql.String("")),
				// comment by format: `attempt <task.Attempts>: <comment>\n`
				mysql.CONCAT(
					mysql.String("attempt ").
						CONCAT(table.Task.Attempts).
						CONCAT(mysql.String(fmt.Sprintf(": %s\n", comment))),
				),
			),
			mysql.TimestampT(xtime.Now()),
		).
		WHERE(
			table.Task.ID.IN(lo.Map(tasks, func(item *model.Task, _ int) mysql.Expression {
				return mysql.String(item.ID)
			})...),
		)

	query, args := updateStmt.Sql()
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task", slog.Any("err", err))
		return err
	}

	return nil
}
