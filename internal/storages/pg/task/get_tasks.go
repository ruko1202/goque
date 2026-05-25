package task

import (
	"context"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetTasks",
		xfield.Any("filter", filter),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	tasks, err := s.getTasksByFilter(ctx, filter, limit)
	if err != nil {
		return nil, err
	}
	return fromDBModels(ctx, tasks), nil
}

func (s *Storage) getTasksByFilter(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*model.GoqueTask, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.getTasksByFilter")
	defer span.End()

	whereExpr, err := filter.BindPgWhereExpr()
	if err != nil {
		xlog.Error(ctx, "failed to bind filter", xfield.Error(err))
		return nil, err
	}

	stmt := table.GoqueTask.
		SELECT(table.GoqueTask.AllColumns).
		WHERE(whereExpr).
		LIMIT(limit)

	query, args := stmt.Sql()

	tasks := make([]*model.GoqueTask, 0)
	err = s.db.Executor(ctx).SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to get tasks", xfield.Error(err))
		return nil, err
	}

	return tasks, nil
}
