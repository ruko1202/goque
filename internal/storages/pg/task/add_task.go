package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lib/pq"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/storages/dbentity"

	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	dbTask := toDBModel(task)

	stmt := table.Task.
		INSERT(table.Task.AllColumns).
		MODEL(dbTask)

	query, args := stmt.Sql()

	_, err := s.db.ExecContext(ctx, query, args...)
	if err := handleError(err); err != nil {
		slog.ErrorContext(ctx, "failed to add task", slog.Any("err", err))
		return err
	}

	return nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code.Name() {
		case "unique_violation":
			return fmt.Errorf("%w: %s", dbentity.ErrDuplicateTask, pqErr.Detail)
		case "invalid_text_representation":
			return fmt.Errorf("%w: %s", dbentity.ErrInvalidPayloadFormat, pqErr.Detail)
		default:
			return pqErr
		}
	}

	return err
}
