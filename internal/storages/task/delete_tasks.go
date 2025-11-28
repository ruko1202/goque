package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// DeleteTasks removes tasks with specified statuses that haven't been updated within the given time period.
func (s *Storage) DeleteTasks(ctx context.Context, statuses []entity.TaskStatus, updatedAtTimeAgo time.Duration) ([]*entity.Task, error) {
	stmt := table.Task.DELETE().
		WHERE(
			postgres.AND(
				table.Task.Status.IN(lo.Map(statuses, func(status string, _ int) postgres.Expression {
					return postgres.String(status)
				})...),
				table.Task.UpdatedAt.LT_EQ(
					postgres.NOW().SUB(
						postgres.INTERVALd(updatedAtTimeAgo.Abs()),
					),
				),
			),
		).
		RETURNING(table.Task.AllColumns)

	query, args := stmt.Sql()

	tasks := make([]*entity.Task, 0)
	err := s.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete tasks", slog.Any("err", err))
		return nil, err
	}
	return tasks, nil
}
