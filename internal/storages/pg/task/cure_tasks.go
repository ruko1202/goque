package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/utils/xtime"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// CureTasks updates stuck tasks to error status for retry.
func (s *Storage) CureTasks(ctx context.Context, taskType entity.TaskType, unhealthStatuses []entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error) {
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
				table.Task.Status.IN(lo.Map(unhealthStatuses, func(item entity.TaskStatus, _ int) postgres.Expression {
					return postgres.String(item)
				})...),
				table.Task.UpdatedAt.LT_EQ(
					postgres.TimestampzT(xtime.Now().Add(-updatedAtTimeAgo.Abs())),
				),
			),
		).RETURNING(table.Task.AllColumns)

	query, args := stmt.Sql()

	dbTasks := make([]*model.Task, 0)
	err := s.db.SelectContext(ctx, &dbTasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task", slog.Any("err", err))
		return nil, err
	}

	return fromDBModels(dbTasks), nil
}
