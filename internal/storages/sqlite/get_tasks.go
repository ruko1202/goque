package sqlite

import (
	"context"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetTasks",
		xfield.String("db.type", "sqlite"),
		xfield.Any("filter", filter),
	)
	defer span.End()

	tasks, err := s.getTasksByFilterTx(ctx, s.db, filter, limit)
	if err != nil {
		return nil, err
	}
	return fromDBModels(ctx, tasks)
}

func (s *Storage) getTasksByFilterTx(ctx context.Context, tx dbutils.DBTx, filter *dbentity.GetTasksFilter, limit int64) ([]*model.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.getTasksByFilterTx")
	defer span.End()

	whereExpr, err := filter.BindSqliteWhereExpr()
	if err != nil {
		xlog.Error(ctx, "failed to bind filter", xfield.Error(err))
		return nil, err
	}

	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(whereExpr).
		LIMIT(limit)

	dbTasks := make([]*model.Task, 0)
	err = stmt.QueryContext(ctx, tx, &dbTasks)
	if err != nil {
		xlog.Error(ctx, "failed to get tasks", xfield.Error(err))
		return nil, err
	}

	return dbTasks, nil
}
