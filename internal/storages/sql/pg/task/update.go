package task

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	sqldbutils "github.com/ruko1202/goque/internal/storages/sql/utils"
)

// UpdateTask updates an existing task in the database with the provided data.
func (s *Storage) UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error {
	return s.updateTask(ctx, s.db, taskID, task)
}

func (s *Storage) updateTask(ctx context.Context, tx sqldbutils.DBTx, taskID uuid.UUID, task *entity.Task) error {
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
			postgres.NOW(),
			task.NextAttemptAt,
		).
		WHERE(table.Task.ID.EQ(postgres.UUID(taskID))).
		RETURNING(table.Task.MutableColumns)

	query, args := stmt.Sql()

	err := tx.GetContext(ctx, task, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task", slog.Any("err", err))
		return err
	}

	return nil
}

func (s *Storage) batchUpdateTasksStatusTx(ctx context.Context, tx sqldbutils.DBTx, taskIDs []uuid.UUID, newStatus string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	stmt := table.Task.
		UPDATE(
			table.Task.Status,
			table.Task.UpdatedAt,
		).
		SET(
			postgres.String(newStatus),
			postgres.NOW(),
		).
		WHERE(table.Task.ID.IN(
			lo.Map(taskIDs, func(taskID uuid.UUID, _ int) postgres.Expression {
				return postgres.UUID(taskID)
			})...,
		))

	query, args := stmt.Sql()

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task", slog.Any("err", err))
		return err
	}

	return nil
}
