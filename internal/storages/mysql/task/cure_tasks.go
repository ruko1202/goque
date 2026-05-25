package mysqltask

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// CureTasks updates stuck tasks to error status for retry.
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
	statuses []entity.TaskStatus,
	updatedAtTimeAgo time.Duration,
	comment string,
) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.CureTasks",
		xfield.String("db.type", "mysql"),
		xfield.Any("statuses", statuses),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	defer span.End()

	tasks := make([]*model.GoqueTask, 0)
	err := dbtx.WithinTx(ctx, s.db.GetDB(), func(ctx context.Context) error {
		var err error
		// FOR UPDATE SKIP LOCKED: a concurrent CureTasks tick on the
		// same task_type will see a different (non-overlapping) slice
		// of rows. Without this lock the SELECT/UPDATE pair has a race
		// where two workers both see and both cure the same task.
		tasks, err = s.getTasksByFilterForUpdate(ctx, &dbentity.GetTasksFilter{
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
	updateStmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Errors,
			table.GoqueTask.UpdatedAt,
		).
		SET(
			mysql.String(entity.TaskStatusError),
			mysql.CONCAT(
				mysql.COALESCE(table.GoqueTask.Errors, mysql.String("")),
				// comment by format: `attempt <task.Attempts>: <comment>\n`
				mysql.CONCAT(
					mysql.String("attempt ").
						CONCAT(table.GoqueTask.Attempts).
						CONCAT(mysql.String(fmt.Sprintf(": %s\n", comment))),
				),
			),
			mysql.TimestampT(xtime.Now()),
		).
		WHERE(
			table.GoqueTask.ID.IN(lo.Map(tasks, func(item *model.GoqueTask, _ int) mysql.Expression {
				return mysql.String(item.ID)
			})...),
		)

	query, args := updateStmt.Sql()
	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
