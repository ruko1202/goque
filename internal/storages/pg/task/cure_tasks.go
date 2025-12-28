package task

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks updates stuck tasks to error status for retry.
func (s *Storage) CureTasks(
	ctx context.Context,
	taskType entity.TaskType,
	statuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.CureTasks",
		zap.Any("statuses", statuses),
		zap.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)

	stmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.Errors,
			table.Task.UpdatedAt,
		).
		SET(
			postgres.String(entity.TaskStatusError),
			postgres.CONCAT(
				postgres.COALESCE(table.Task.Errors, postgres.String("")),
				// commen by format: `attempt <task.Attempts>: <comment>\n`
				postgres.String("attempt ").
					CONCAT(table.Task.Attempts).
					CONCAT(postgres.String(fmt.Sprintf(": %s\n", comment))),
			),
			postgres.TimestampzT(xtime.Now()),
		).
		WHERE(
			postgres.AND(
				table.Task.Type.EQ(postgres.String(taskType)),
				table.Task.Status.IN(lo.Map(statuses, func(item entity.TaskStatus, _ int) postgres.Expression {
					return postgres.String(item)
				})...),
				table.Task.UpdatedAt.LT_EQ(
					postgres.TimestampzT(xtime.Now().Add(-updatedAtTimeAgo.Abs())),
				),
			),
		).RETURNING(table.Task.AllColumns)

	dbTasks := make([]*model.Task, 0)
	err := stmt.QueryContext(ctx, s.db, &dbTasks)
	if err != nil {
		xlog.Error(ctx, "failed to update task", zap.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, dbTasks), nil
}
