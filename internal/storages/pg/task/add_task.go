package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.AddTask",
		xfield.String("task_id", task.ID.String()),
		xfield.String("task_type", task.Type),
	)
	span.SetAttributes(semconv.DBSystemNamePostgreSQL)
	defer span.End()

	// Validate JSON payload before insertion
	if !dbutils.IsValidJSON(task.Payload) {
		return entity.ErrInvalidPayloadFormat
	}
	dbTask := toDBModel(ctx, task)

	stmt := table.GoqueTask.
		INSERT(table.GoqueTask.AllColumns).
		MODEL(dbTask)

	query, args := stmt.Sql()

	_, err := s.db.Executor(ctx).ExecContext(ctx, query, args...)
	if err := handleError(err); err != nil {
		xlog.Error(ctx, "failed to add task", xfield.Error(err))
		return err
	}

	return nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	// pgx returns *pgconn.PgError for server-side Postgres errors.
	// pgErr.Code is the SQLSTATE string ("23505" etc.); use the
	// pgerrcode named constants for readability.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return fmt.Errorf("%w: %s", entity.ErrDuplicateTask, pgErr.Detail)
		case pgerrcode.InvalidTextRepresentation:
			return fmt.Errorf("%w: %s", entity.ErrInvalidPayloadFormat, pgErr.Detail)
		default:
			return pgErr
		}
	}

	return err
}
