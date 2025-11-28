package mysqltask

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-sql-driver/mysql"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
	"github.com/ruko1202/goque/internal/storages/dbentity"
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

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062: // ER_DUP_ENTRY
			return fmt.Errorf("%w: %s", dbentity.ErrDuplicateTask, mysqlErr.Message)
		case 3140, 3141, 3142: // JSON_INVALID_DATA
			return fmt.Errorf("%w: %s", dbentity.ErrInvalidPayloadFormat, mysqlErr.Message)
		default:
			return mysqlErr
		}
	}

	return err
}
