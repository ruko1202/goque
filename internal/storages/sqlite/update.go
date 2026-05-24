package sqlite

import (
	"context"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// UpdateTask updates an existing task in the database with the provided data.
func (s *Storage) UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.UpdateTask",
		xfield.String("db.type", "sqlite"),
		xfield.String("task_id", taskID.String()),
	)
	defer span.End()

	task.UpdatedAt = lo.ToPtr(xtime.Now())
	return s.updateTaskTx(ctx, s.db, taskID.String(), toDBModel(ctx, task))
}

// HardUpdateTask updates a task without automatically setting the updated_at timestamp.
func (s *Storage) HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.HardUpdateTask",
		xfield.String("db.type", "sqlite"),
		xfield.String("task_id", taskID.String()),
	)
	defer span.End()

	return s.updateTaskTx(ctx, s.db, taskID.String(), toDBModel(ctx, task))
}

func (s *Storage) updateTaskTx(ctx context.Context, tx dbutils.DBTx, taskID string, task *model.GoqueTask) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.updateTaskTx")
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
		WHERE(table.GoqueTask.ID.EQ(sqlite.String(taskID)))

	query, args := stmt.Sql()

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", xfield.Error(err))
		return err
	}

	return nil
}

func (s *Storage) batchUpdateTasksStatusTx(ctx context.Context, tx dbutils.DBTx, tasks []*model.GoqueTask, newStatus string) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.batchUpdateTasksStatusTx")
	defer span.End()

	if len(tasks) == 0 {
		return nil
	}

	now := timeToString(xtime.Now())
	stmt := table.GoqueTask.
		UPDATE(
			table.GoqueTask.Status,
			table.GoqueTask.UpdatedAt,
		).
		SET(
			sqlite.String(newStatus),
			sqlite.String(now),
		).
		WHERE(table.GoqueTask.ID.IN(
			lo.Map(tasks, func(task *model.GoqueTask, _ int) sqlite.Expression { return sqlite.String(lo.FromPtr(task.ID)) })...,
		))

	query, args := stmt.Sql()

	_, err := tx.ExecContext(ctx, query, args...)
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
