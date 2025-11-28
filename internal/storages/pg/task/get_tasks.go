package task

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/storages/dbutils"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	tasks, err := s.getTasksByFilterTx(ctx, s.db, filter, limit)
	if err != nil {
		return nil, err
	}
	return fromDBModels(tasks), nil
}

func (s *Storage) getTasksByFilterTx(ctx context.Context, tx dbutils.DBTx, filter *dbentity.GetTasksFilter, limit int64) ([]*model.Task, error) {
	whereExpr, err := filter.BindMysqlWhereExpr()
	if err != nil {
		slog.ErrorContext(ctx, "failed to bind filter", slog.Any("err", err))
		return nil, err
	}

	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(whereExpr).
		LIMIT(limit)

	query, args := stmt.Sql()

	tasks := make([]*model.Task, 0)
	err = tx.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get tasks", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
