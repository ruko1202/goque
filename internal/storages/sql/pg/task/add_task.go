package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lib/pq"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
)

var (
	// ErrDuplicateTask is returned when attempting to insert a task with a duplicate external ID.
	ErrDuplicateTask = errors.New("task already exists")
	// ErrInvalidPayloadFormat is returned when the task payload is not valid JSON.
	ErrInvalidPayloadFormat = errors.New("payload format is invalid. should be json")
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
			return fmt.Errorf("%w: %s", ErrDuplicateTask, pqErr.Detail)
		case "invalid_text_representation":
			return fmt.Errorf("%w: %s", ErrInvalidPayloadFormat, pqErr.Detail)
		default:
			return pqErr
		}
	}

	return err
}
