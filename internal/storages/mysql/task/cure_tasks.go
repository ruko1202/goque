package mysqltask

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks moves tasks stuck in status to error and bumps their version.
// A task cured from processing may have run, so it also consumes an attempt
// and gets the comment appended to its errors; a task cured from pending never
// started and keeps both.
//
// MySQL has no UPDATE ... RETURNING, so we can't atomically update
// rows and read them back in a single statement (PG storage does
// exactly that). Instead we run SELECT ... FOR UPDATE SKIP LOCKED
// inside a transaction, then UPDATE WHERE id IN (...). The row-level
// locks taken by the SELECT close the race window where two healers
// would otherwise see and cure the same task; SKIP_LOCKED keeps the
// second healer from blocking — it just picks up a different slice
// of rows on its tick.
//
// Cost is one extra roundtrip vs. PG. There is no way around it
// without breaking the HealerTaskStorage API (callers consume the
// returned []*entity.Task for logs + metrics).
func (s *Storage) CureTasks(
	ctx context.Context,
	taskType entity.TaskType,
	status entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.CureTasks",
		xfield.String("db.type", "mysql"),
		xfield.String("status", status),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	defer span.End()

	tasks := make([]*model.GoqueTask, 0)
	err := s.withinTx(ctx, func(ctx context.Context) error {
		var err error
		// FOR UPDATE SKIP LOCKED: a concurrent CureTasks tick on the
		// same task_type will see a different (non-overlapping) slice
		// of rows. Without this lock the SELECT/UPDATE pair has a race
		// where two workers both see and both cure the same task.
		tasks, err = s.getTasksByFilterForUpdate(ctx, &dbentity.GetTasksFilter{
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
	// errors is assigned before attempts: MySQL evaluates SET left to right, and
	// the comment must carry the attempt number before the cure.
	var errs, attempts mysql.Expression = table.GoqueTask.Errors, table.GoqueTask.Attempts
	if consumeAttempt {
		errs = mysql.CONCAT(
			mysql.COALESCE(table.GoqueTask.Errors, mysql.String("")),
			// comment by format: `attempt <task.Attempts>: <comment>\n`
			mysql.String("attempt ").
				CONCAT(table.GoqueTask.Attempts).
				CONCAT(mysql.String(fmt.Sprintf(": %s\n", comment))),
		)
		attempts = table.GoqueTask.Attempts.ADD(mysql.Int(1))
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
			mysql.String(entity.TaskStatusError),
			errs,
			attempts,
			mysql.TimestampT(now),
			// Invalidates the version held by the task's current holder.
			table.GoqueTask.Version.ADD(mysql.Int(1)),
		).
		WHERE(mysql.OR(
			lo.Map(tasks, func(item *model.GoqueTask, _ int) mysql.BoolExpression {
				return mysql.AND(
					table.GoqueTask.ID.EQ(mysql.String(item.ID)),
					table.GoqueTask.Version.EQ(mysql.Int(item.Version)),
				)
			})...,
		))

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
		item.UpdatedAt = lo.ToPtr(now)
		item.Version++
	})

	return nil
}
