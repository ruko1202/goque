package mysqltask

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/google/uuid"

	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
)

// GetTask retrieves a single task by its ID from the database.
func (s *Storage) GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(table.Task.ID.EQ(mysql.String(id.String())))

	query, args := stmt.Sql()

	task := new(model.Task)
	err := s.db.GetContext(ctx, task, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task", slog.Any("err", err))
		return nil, err
	}

	return fromDBModel(task)
}
