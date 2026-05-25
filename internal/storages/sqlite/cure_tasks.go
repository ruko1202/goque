package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks updates stuck tasks to error status for retry.
func (s *Storage) CureTasks(
	ctx context.Context,
	taskType entity.TaskType,
	statuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.CureTasks",
		xfield.String("db.type", "sqlite"),
		xfield.Any("statuses", statuses),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	defer span.End()

	tasks := make([]*model.GoqueTask, 0)
	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		var err error
		tasks, err = s.getTasksByFilter(ctx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         statuses,
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			return err
		}

		return s.cureTask(ctx, tasks, comment)
	})
	if err != nil {
		xlog.Error(ctx, "failed to cure tasks", xfield.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks)
}

func (s *Storage) cureTask(ctx context.Context, tasks []*model.GoqueTask, comment string) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.cureTask")
	defer span.End()

	if len(tasks) == 0 {
		return nil
	}

	// SQLite uses || for concatenation and CAST(x AS TEXT) for type conversion
	updateStmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Errors,
			table.GoqueTask.UpdatedAt,
		).
		SET(
			sqlite.String(entity.TaskStatusError),
			// In SQLite: '' || COALESCE(errors, '') || 'attempt ' || <task.Attempts> || ': <comment>\n'
			sqlite.String("").
				CONCAT(sqlite.COALESCE(table.GoqueTask.Errors, sqlite.String(""))).
				//comment by format: `attempt <task.Attempts>: <comment>\n`
				CONCAT(sqlite.String("attempt ")).
				CONCAT(table.GoqueTask.Attempts).
				CONCAT(sqlite.String(fmt.Sprintf(": %s\n", comment))),
			sqlite.String(timeToString(xtime.Now())),
		).
		WHERE(
			table.GoqueTask.ID.IN(lo.Map(tasks, func(item *model.GoqueTask, _ int) sqlite.Expression {
				return sqlite.String(lo.FromPtr(item.ID))
			})...),
		)

	query, args := updateStmt.Sql()
	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
