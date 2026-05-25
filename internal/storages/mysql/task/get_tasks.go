package mysqltask

import (
	"context"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.GetTasks",
		xfield.String("db.type", "mysql"),
		xfield.Any("filter", filter),
	)
	defer span.End()

	tasks, err := s.getTasksByFilter(ctx, filter, limit)
	if err != nil {
		return nil, err
	}
	return fromDBModels(ctx, tasks)
}

func (s *Storage) getTasksByFilter(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*model.GoqueTask, error) {
	return s.selectTasksByFilter(ctx, filter, limit, false)
}

// getTasksByFilterForUpdate is the same as getTasksByFilter but appends
// FOR UPDATE SKIP LOCKED so concurrent CureTasks/DeleteTasks workers
// don't see the same rows. Must be called inside a transaction —
// MySQL releases row locks at commit/rollback.
func (s *Storage) getTasksByFilterForUpdate(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*model.GoqueTask, error) {
	return s.selectTasksByFilter(ctx, filter, limit, true)
}

func (s *Storage) selectTasksByFilter(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64, forUpdate bool) ([]*model.GoqueTask, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.getTasksByFilter")
	defer span.End()

	whereExpr, err := filter.BindMysqlWhereExpr()
	if err != nil {
		xlog.Error(ctx, "failed to bind filter", xfield.Error(err))
		return nil, err
	}

	stmt := table.GoqueTask.
		SELECT(table.GoqueTask.AllColumns).
		WHERE(whereExpr).
		LIMIT(limit)

	if forUpdate {
		// SKIP LOCKED: other workers' concurrent CureTasks scans
		// silently skip these rows instead of blocking — turns the
		// race into a "next tick will get them" non-issue.
		stmt = stmt.FOR(mysql.UPDATE().SKIP_LOCKED())
	}

	query, args := stmt.Sql()

	tasks := make([]*model.GoqueTask, 0)
	err = s.db.Executor(ctx).SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to get tasks", xfield.Error(err))
		return nil, err
	}

	return tasks, nil
}
