package task

import (
	"context"
	"fmt"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// UpdateTask updates an existing task in the database with the provided data.
func (s *Storage) UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.UpdateTask",
		xfield.String("task_id", taskID.String()),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	task.UpdatedAt = lo.ToPtr(xtime.Now())
	if err := s.updateTask(ctx, taskID, toDBModel(ctx, task), true); err != nil {
		return err
	}
	task.Version++

	return nil
}

// HardUpdateTask updates a task without automatically setting the updated_at timestamp.
func (s *Storage) HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.HardUpdateTask",
		xfield.String("task_id", taskID.String()),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	return s.updateTask(ctx, taskID, toDBModel(ctx, task), false)
}

// updateTask writes the task. When fenced, the write applies only if the stored
// version still equals task.Version and bumps the stored version; a mismatch (or
// a deleted row) returns entity.ErrTaskVersionConflict. Unfenced writes leave the
// version alone and are used only by the test-only HardUpdateTask.
func (s *Storage) updateTask(ctx context.Context, taskID uuid.UUID, task *model.GoqueTask, fenced bool) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.updateTask")
	defer span.End()

	// Unfenced writes keep the version as is (version = version).
	var version postgres.Expression = table.GoqueTask.Version
	where := table.GoqueTask.ID.EQ(postgres.UUID(taskID))
	if fenced {
		version = table.GoqueTask.Version.ADD(postgres.Int(1))
		where = where.AND(table.GoqueTask.Version.EQ(postgres.Int(task.Version)))
	}

	stmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Attempts,
			table.GoqueTask.Errors,
			table.GoqueTask.UpdatedAt,
			table.GoqueTask.NextAttemptAt,
			table.GoqueTask.Version,
		).
		SET(
			task.Status,
			task.Attempts,
			task.Errors,
			task.UpdatedAt,
			task.NextAttemptAt,
			version,
		).
		WHERE(where)

	query, args := stmt.Sql()

	res, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return err
	}
	if !fenced {
		return nil
	}

	return dbutils.ExpectRowsAffected(res, 1, fmt.Sprintf("task %s, expected version %d", taskID, task.Version))
}

func (s *Storage) batchUpdateTasksStatus(ctx context.Context, tasks []*model.GoqueTask, newStatus string) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.batchUpdateTasksStatus")
	defer span.End()

	if len(tasks) == 0 {
		return nil
	}
	now := xtime.Now()
	stmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.UpdatedAt,
			table.GoqueTask.Version,
		).
		SET(
			postgres.String(newStatus),
			postgres.TimestampzT(now),
			table.GoqueTask.Version.ADD(postgres.Int(1)),
		).
		WHERE(postgres.OR(
			lo.Map(tasks, func(task *model.GoqueTask, _ int) postgres.BoolExpression {
				return postgres.AND(
					table.GoqueTask.ID.EQ(postgres.UUID(task.ID)),
					table.GoqueTask.Version.EQ(postgres.Int(task.Version)),
				)
			})...,
		))

	query, args := stmt.Sql()

	res, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return err
	}
	// Every selected row must still have the version it was read with;
	// otherwise the caller's tx rolls back and nothing is fetched.
	if err := dbutils.ExpectRowsAffected(res, int64(len(tasks)), "batch status update"); err != nil {
		return err
	}

	lo.ForEach(tasks, func(task *model.GoqueTask, _ int) {
		task.UpdatedAt = lo.ToPtr(now)
		task.Status = newStatus
		task.Version++
	})

	return nil
}
