package task

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// GetTask retrieves a task from the database by its ID.
func (s *Storage) GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(table.Task.ID.EQ(postgres.UUID(id)))

	query, args := stmt.Sql()

	task := &entity.Task{}

	err := s.db.GetContext(ctx, task, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task", slog.Any("err", err))
		return nil, err
	}

	return task, nil
}
