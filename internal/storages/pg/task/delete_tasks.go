package task

import (
	"context"
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

	stmt := table.Task.DELETE().
		WHERE(
			postgres.AND(
				table.Task.Type.EQ(postgres.String(taskType)),
				table.Task.Status.IN(lo.Map(statuses, func(status string, _ int) postgres.Expression {
					return postgres.String(status)
				})...),
				table.Task.UpdatedAt.LT_EQ(
					postgres.TimestampzT(xtime.Now().Add(-updatedAtTimeAgo.Abs())),
				),
			),
		).
		RETURNING(table.Task.AllColumns)

	dbTasks := make([]*model.Task, 0)
	err := stmt.QueryContext(ctx, s.db, &dbTasks)
	if err != nil {
		xlog.Error(ctx, "failed to delete tasks", zap.Error(err))
		return nil, err
	}
	return fromDBModels(ctx, dbTasks), nil
}
