package task

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbutils"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// GetTasksFilter defines filtering criteria for task queries.
type GetTasksFilter struct {
	TaskType entity.TaskType
	Status   *entity.TaskStatus
}

func (f *GetTasksFilter) bindWhereExpr() postgres.BoolExpression {
	expr := postgres.AND(
		table.Task.Type.EQ(postgres.String(f.TaskType)),
	)

	if f.Status != nil {
		expr = expr.AND(
			table.Task.Status.EQ(postgres.String(lo.FromPtr(f.Status))),
		)
	}

	return expr
}

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *GetTasksFilter, limit int64) ([]*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(filter.bindWhereExpr()).
		LIMIT(limit)

	return s.getTasksTx(ctx, s.db, stmt)
}

// GetOlderTasks retrieves tasks ordered by creation date (oldest first) matching the filter.
func (s *Storage) GetOlderTasks(ctx context.Context, filter *GetTasksFilter, limit int64) ([]*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(filter.bindWhereExpr()).
		ORDER_BY(table.Task.CreatedAt.ASC()).
		LIMIT(limit)

	return s.getTasksTx(ctx, s.db, stmt)
}

// GetNewestTasks retrieves tasks ordered by creation date (newest first) matching the filter.
func (s *Storage) GetNewestTasks(ctx context.Context, filter *GetTasksFilter, limit int64) ([]*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(filter.bindWhereExpr()).
		ORDER_BY(table.Task.CreatedAt.DESC()).
		LIMIT(limit)

	return s.getTasksTx(ctx, s.db, stmt)
}

func (s *Storage) getTasksTx(ctx context.Context, tx dbutils.DBTx, stmt postgres.Statement) ([]*entity.Task, error) {
	query, args := stmt.Sql()

	tasks := make([]*entity.Task, 0)
	err := tx.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get tasks", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
