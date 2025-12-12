package sqlite

import (
	"context"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// UpdateTask updates an existing task in the database with the provided data.
func (s *Storage) UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	task.UpdatedAt = lo.ToPtr(xtime.Now())
	return s.updateTaskTx(ctx, s.db, taskID.String(), toDBModel(ctx, task))
}

// HardUpdateTask updates a task without automatically setting the updated_at timestamp.
func (s *Storage) HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	return s.updateTaskTx(ctx, s.db, taskID.String(), toDBModel(ctx, task))
}

func (s *Storage) updateTaskTx(ctx context.Context, tx dbutils.DBTx, taskID string, task *model.Task) error {
	ctx = xlog.WithOperation(ctx, "storage.UpdateTask",
		zap.String("task_id", taskID),
	)

	stmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.Attempts,
			table.Task.Errors,
			table.Task.UpdatedAt,
			table.Task.NextAttemptAt,
		).
		SET(
			task.Status,
			task.Attempts,
			task.Errors,
			task.UpdatedAt,
			task.NextAttemptAt,
		).
		WHERE(table.Task.ID.EQ(sqlite.String(taskID)))

	query, args := stmt.Sql()

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", zap.Error(err))
		return err
	}

	return nil
}

func (s *Storage) batchUpdateTasksStatusTx(ctx context.Context, tx dbutils.DBTx, tasks []*model.Task, newStatus string) error {
	if len(tasks) == 0 {
		return nil
	}

	now := timeToString(xtime.Now())
	stmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.UpdatedAt,
		).
		SET(
			sqlite.String(newStatus),
			sqlite.String(now),
		).
		WHERE(table.Task.ID.IN(
			lo.Map(tasks, func(task *model.Task, _ int) sqlite.Expression { return sqlite.String(lo.FromPtr(task.ID)) })...,
		))

	query, args := stmt.Sql()

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to update task", zap.Error(err))
		return err
	}

	lo.ForEach(tasks, func(task *model.Task, _ int) {
		task.UpdatedAt = lo.ToPtr(now)
		task.Status = newStatus
	})

	return nil
}
