package task

import (
	"context"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
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
	return s.updateTask(ctx, taskID, toDBModel(ctx, task))
}

// HardUpdateTask updates a task without automatically setting the updated_at timestamp.
func (s *Storage) HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.HardUpdateTask",
		xfield.String("task_id", taskID.String()),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	return s.updateTask(ctx, taskID, toDBModel(ctx, task))
}

func (s *Storage) updateTask(ctx context.Context, taskID uuid.UUID, task *model.GoqueTask) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.updateTask")
	defer span.End()

	stmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.Attempts,
			table.GoqueTask.Errors,
			table.GoqueTask.UpdatedAt,
			table.GoqueTask.NextAttemptAt,
		).
		SET(
			task.Status,
			task.Attempts,
			task.Errors,
			task.UpdatedAt,
			task.NextAttemptAt,
		).
		WHERE(table.GoqueTask.ID.EQ(postgres.UUID(taskID)))

	query, args := stmt.Sql()

	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return err
	}

	return nil
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
		).
		SET(
			postgres.String(newStatus),
			postgres.TimestampzT(now),
		).
		WHERE(table.GoqueTask.ID.IN(
			lo.Map(tasks, func(task *model.GoqueTask, _ int) postgres.Expression {
				return postgres.UUID(task.ID)
			})...,
		)).
		RETURNING(table.GoqueTask.MutableColumns)

	query, args := stmt.Sql()

	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return err
	}

	lo.ForEach(tasks, func(task *model.GoqueTask, _ int) {
		task.UpdatedAt = lo.ToPtr(now)
		task.Status = newStatus
	})

	return nil
}
