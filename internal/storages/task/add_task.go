package task

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	stmt := table.Task.
		INSERT(table.Task.AllColumns.
			Except(
				table.Task.DefaultColumns,
			),
		).
		MODEL(task).
		RETURNING(table.Task.DefaultColumns)

	query, args := stmt.Sql()

	err := s.db.GetContext(ctx, task, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to add task", slog.Any("err", err))
		return err
	}

	return nil
}
