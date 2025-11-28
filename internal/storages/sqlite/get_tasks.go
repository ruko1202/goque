package sqlite

import (
	"context"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.GetTasks",
		zap.Any("filter", filter),
	)

	tasks, err := s.getTasksByFilterTx(ctx, s.db, filter, limit)
	if err != nil {
		return nil, err
	}
	return fromDBModels(tasks)
}

func (s *Storage) getTasksByFilterTx(ctx context.Context, tx dbutils.DBTx, filter *dbentity.GetTasksFilter, limit int64) ([]*model.Task, error) {
	whereExpr, err := filter.BindSqliteWhereExpr()
	if err != nil {
		xlog.Error(ctx, "failed to bind filter", zap.Error(err))
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
		xlog.Error(ctx, "failed to get tasks", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}
