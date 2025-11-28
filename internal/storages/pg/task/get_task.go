package task

import (
	"context"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// GetTask retrieves a single task by its ID from the database.
func (s *Storage) GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.GetTask")

	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(table.Task.ID.EQ(postgres.UUID(id)))

	query, args := stmt.Sql()

	task := new(model.Task)
	err := s.db.GetContext(ctx, task, query, args...)
	if err != nil {
		xlog.Error(ctx, "failed to get task", zap.Error(err))
		return nil, err
	}

	return fromDBModel(task), nil
}
