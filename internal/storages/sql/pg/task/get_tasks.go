package task

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	sqlpgutils "github.com/ruko1202/goque/internal/storages/sql/pg/utils"
	sqldbutils "github.com/ruko1202/goque/internal/storages/sql/utils"
)

// GetTasksFilter defines filtering criteria for task queries.
type GetTasksFilter struct {
	TaskType         *entity.TaskType
	Status           *entity.TaskStatus
	UpdatedAtTimeAgo *time.Duration
}

func (f *GetTasksFilter) bindWhereExpr() (postgres.BoolExpression, error) {
	expr := sqlpgutils.NewWhereBuilder()

	if f.TaskType != nil {
		expr.And(
			table.Task.Type.EQ(postgres.String(lo.FromPtr(f.TaskType))),
		)
	}

	if f.Status != nil {
		expr.And(
			table.Task.Status.EQ(postgres.String(lo.FromPtr(f.Status))),
		)
	}

	if f.UpdatedAtTimeAgo != nil {
		expr.And(
			table.Task.UpdatedAt.LT_EQ(
				postgres.NOW().SUB(
					postgres.INTERVALd(f.UpdatedAtTimeAgo.Abs()),
				),
			),
		)
	}

	if expr == nil {
		return nil, errors.New("no filter criteria specified")
	}

	return expr.Expression(), nil
}

// GetTasks retrieves tasks matching the filter criteria with a specified limit.
func (s *Storage) GetTasks(ctx context.Context, filter *GetTasksFilter, limit int64) ([]*entity.Task, error) {
	whereExpr, err := filter.bindWhereExpr()
	if err != nil {
		slog.ErrorContext(ctx, "failed to bind filter", slog.Any("err", err))
		return nil, err
	}

	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(whereExpr).
		LIMIT(limit)

	return s.getTasksTx(ctx, s.db, stmt, false)
}

func (s *Storage) getTasksTx(ctx context.Context, tx sqldbutils.DBTx, stmt postgres.SelectStatement, forUpdate bool) ([]*entity.Task, error) {
	if forUpdate {
		stmt = stmt.FOR(postgres.UPDATE())
	}
	query, args := stmt.Sql()

	tasks := make([]*entity.Task, 0)
	err := tx.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get tasks", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
