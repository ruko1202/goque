package task

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
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
		xfield.String("status", status),
		xfield.Duration("updated_at_time_ago", updatedAtTimeAgo),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	// A task cured from pending never started: errors and attempts stay as is.
	var errs, attempts postgres.Expression = table.GoqueTask.Errors, table.GoqueTask.Attempts
	if status == entity.TaskStatusProcessing {
		errs = postgres.CONCAT(
			postgres.COALESCE(table.GoqueTask.Errors, postgres.String("")),
			// comment by format: `attempt <task.Attempts>: <comment>\n`
			postgres.String("attempt ").
				CONCAT(table.GoqueTask.Attempts).
				CONCAT(postgres.String(fmt.Sprintf(": %s\n", comment))),
		)
		attempts = table.GoqueTask.Attempts.ADD(postgres.Int(1))
	}

	stmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Errors,
			table.GoqueTask.Attempts,
			table.GoqueTask.UpdatedAt,
			table.GoqueTask.Version,
		).
		SET(
			postgres.String(entity.TaskStatusError),
			errs,
			attempts,
			postgres.TimestampzT(xtime.Now()),
			// Invalidates the version held by the task's current holder.
			table.GoqueTask.Version.ADD(postgres.Int(1)),
		).
		WHERE(
			postgres.AND(
				table.GoqueTask.Type.EQ(postgres.String(taskType)),
				table.GoqueTask.Status.EQ(postgres.String(status)),
				table.GoqueTask.UpdatedAt.LT_EQ(
					postgres.TimestampzT(xtime.Now().Add(-updatedAtTimeAgo.Abs())),
				),
			),
		).RETURNING(table.GoqueTask.AllColumns)

	query, args := stmt.Sql()

	dbTasks := make([]*model.GoqueTask, 0)
	err := s.db.Executor(ctx).SelectContext(ctx, &dbTasks, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return nil, err
	}

	return fromDBModels(ctx, dbTasks), nil
}
