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
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks moves tasks stuck in status to error and bumps their version.
// A task cured from processing may have run, so it also consumes an attempt
// and gets the comment appended to its errors; a task cured from pending never
// started and keeps both.
func (s *Storage) CureTasks(
	ctx context.Context,
	taskType entity.TaskType,
	status entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.CureTasks",
		xfield.String("db.type", "sqlite"),
		xfield.String("status", status),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	defer span.End()

	tasks := make([]*model.GoqueTask, 0)
	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		var err error
		tasks, err = s.getTasksByFilter(ctx, &dbentity.GetTasksFilter{
			TaskType:         lo.ToPtr(taskType),
			Statuses:         []entity.TaskStatus{status},
			UpdatedAtTimeAgo: lo.ToPtr(updatedAtTimeAgo),
		}, 1000)
		if err != nil {
			return err
		}

		return s.cureTask(ctx, tasks, comment, status == entity.TaskStatusProcessing)
	})
	if err != nil {
		xlog.Error(ctx, "failed to cure tasks", xfield.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, tasks)
}

func (s *Storage) cureTask(ctx context.Context, tasks []*model.GoqueTask, comment string, consumeAttempt bool) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.cureTask")
	defer span.End()

	if len(tasks) == 0 {
		return nil
	}

	now := xtime.Now()
	// A task cured from pending never started: errors and attempts stay as is.
	// The comment carries the attempt number before the cure.
	var errs, attempts sqlite.Expression = table.GoqueTask.Errors, table.GoqueTask.Attempts
	if consumeAttempt {
		// In SQLite: '' || COALESCE(errors, '') || 'attempt ' || <task.Attempts> || ': <comment>\n'
		errs = sqlite.String("").
			CONCAT(sqlite.COALESCE(table.GoqueTask.Errors, sqlite.String(""))).
			CONCAT(sqlite.String("attempt ")).
			CONCAT(table.GoqueTask.Attempts).
			CONCAT(sqlite.String(fmt.Sprintf(": %s\n", comment)))
		attempts = table.GoqueTask.Attempts.ADD(sqlite.Int(1))
	}

	updateStmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Errors,
			table.GoqueTask.Attempts,
			table.GoqueTask.UpdatedAt,
			table.GoqueTask.Version,
		).
		SET(
			sqlite.String(entity.TaskStatusError),
			errs,
			attempts,
			sqlite.String(timeToString(now)),
			// Invalidates the version held by the task's current holder.
			table.GoqueTask.Version.ADD(sqlite.Int(1)),
		).
		WHERE(versionPredicate(tasks))

	query, args := updateStmt.Sql()
	res, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if err := dbutils.ExpectRowsAffected(res, int64(len(tasks)), "cure tasks"); err != nil {
		return err
	}

	// Mirror the update in the returned models.
	lo.ForEach(tasks, func(item *model.GoqueTask, _ int) {
		if consumeAttempt {
			errs := lo.FromPtr(item.Errors) + fmt.Sprintf("attempt %d: %s\n", item.Attempts, comment)
			item.Errors = &errs
			item.Attempts++
		}
		item.Status = entity.TaskStatusError
		item.UpdatedAt = lo.ToPtr(timeToString(now))
		item.Version++
	})

	return nil
}
