// Package dbentity provides common database entities and filters for task storage implementations.
package dbentity

import (
	"errors"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"

	mysqltable "github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	pgtable "github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	sqlitetable "github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// GetTasksFilter defines filtering criteria for task queries.
type GetTasksFilter struct {
	IDs              []uuid.UUID
	TaskType         *entity.TaskType
	Status           *entity.TaskStatus
	Statuses         []entity.TaskStatus
	UpdatedAtTimeAgo *time.Duration
}

// BindPgWhereExpr converts the filter to a PostgreSQL WHERE expression using go-jet.
//
//nolint:dupl // Similar to BindMysqlWhereExpr but uses PostgreSQL-specific types, cannot be easily abstracted
func (f *GetTasksFilter) BindPgWhereExpr() (postgres.BoolExpression, error) {
	expr := dbutils.NewPgWhereBuilder()

	if len(f.IDs) > 0 {
		expr.And(
			pgtable.GoqueTask.ID.IN(lo.Map(f.IDs, func(item uuid.UUID, _ int) postgres.Expression {
				return postgres.UUID(item)
			})...),
		)
	}

	if f.TaskType != nil {
		expr.And(
			pgtable.GoqueTask.Type.EQ(postgres.String(lo.FromPtr(f.TaskType))),
		)
	}

	if f.Status != nil {
		expr.And(
			pgtable.GoqueTask.Status.EQ(postgres.String(lo.FromPtr(f.Status))),
		)
	}

	if len(f.Statuses) > 0 {
		expr.And(
			pgtable.GoqueTask.Status.IN(lo.Map(f.Statuses, func(item entity.TaskStatus, _ int) postgres.Expression {
				return postgres.String(item)
			})...),
		)
	}

	if f.UpdatedAtTimeAgo != nil {
		expr.And(
			pgtable.GoqueTask.UpdatedAt.LT_EQ(
				postgres.TimestampzT(xtime.Now().Add(-f.UpdatedAtTimeAgo.Abs())),
			),
		)
	}

	if expr == nil {
		return nil, errors.New("no filter criteria specified")
	}

	return expr.Expression(), nil
}

// BindMysqlWhereExpr converts the filter to a MySQL WHERE expression using go-jet.
//
//nolint:dupl // Similar to BindPgWhereExpr but uses MySQL-specific types, cannot be easily abstracted
func (f *GetTasksFilter) BindMysqlWhereExpr() (mysql.BoolExpression, error) {
	expr := dbutils.NewMysqlWhereBuilder()

	if len(f.IDs) > 0 {
		expr.And(
			mysqltable.GoqueTask.ID.IN(lo.Map(f.IDs, func(item uuid.UUID, _ int) mysql.Expression {
				return mysql.UUID(item)
			})...),
		)
	}
	if f.TaskType != nil {
		expr.And(
			mysqltable.GoqueTask.Type.EQ(mysql.String(lo.FromPtr(f.TaskType))),
		)
	}

	if f.Status != nil {
		expr.And(
			mysqltable.GoqueTask.Status.EQ(mysql.String(lo.FromPtr(f.Status))),
		)
	}

	if len(f.Statuses) > 0 {
		expr.And(
			mysqltable.GoqueTask.Status.IN(lo.Map(f.Statuses, func(item entity.TaskStatus, _ int) mysql.Expression {
				return mysql.String(item)
			})...),
		)
	}

	if f.UpdatedAtTimeAgo != nil {
		expr.And(
			mysqltable.GoqueTask.UpdatedAt.LT_EQ(
				mysql.TimestampT(xtime.Now().Add(-f.UpdatedAtTimeAgo.Abs())),
			),
		)
	}

	if expr == nil {
		return nil, errors.New("no filter criteria specified")
	}

	return expr.Expression(), nil
}

// BindSqliteWhereExpr converts the filter to a SQLite WHERE expression using go-jet.
//
//nolint:dupl // Similar to BindPgWhereExpr but uses SQLite-specific types, cannot be easily abstracted
func (f *GetTasksFilter) BindSqliteWhereExpr() (sqlite.BoolExpression, error) {
	expr := dbutils.NewSqliteWhereBuilder()

	if len(f.IDs) > 0 {
		expr.And(
			sqlitetable.GoqueTask.ID.IN(lo.Map(f.IDs, func(item uuid.UUID, _ int) sqlite.Expression {
				return sqlite.UUID(item)
			})...),
		)
	}
	if f.TaskType != nil {
		expr.And(
			sqlitetable.GoqueTask.Type.EQ(sqlite.String(lo.FromPtr(f.TaskType))),
		)
	}

	if f.Status != nil {
		expr.And(
			sqlitetable.GoqueTask.Status.EQ(sqlite.String(lo.FromPtr(f.Status))),
		)
	}

	if len(f.Statuses) > 0 {
		expr.And(
			sqlitetable.GoqueTask.Status.IN(lo.Map(f.Statuses, func(item entity.TaskStatus, _ int) sqlite.Expression {
				return sqlite.String(item)
			})...),
		)
	}

	if f.UpdatedAtTimeAgo != nil {
		expr.And(
			sqlite.DATETIME(sqlitetable.GoqueTask.UpdatedAt).LT_EQ(
				sqlite.DATETIME(xtime.Now().Add(-f.UpdatedAtTimeAgo.Abs())),
			),
		)
	}

	if expr == nil {
		return nil, errors.New("no filter criteria specified")
	}

	return expr.Expression(), nil
}
