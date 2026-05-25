package sqlite

import (
	"context"
	"strings"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

// AddTask inserts a new task into the database.
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "storage.AddTask",
		xfield.String("db.type", "sqlite"),
		xfield.String("task_id", task.ID.String()),
		xfield.String("task_type", task.Type),
	)
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

	// String-match the unique-constraint error message rather than
	// type-asserting on sqlite3.Error. Typed assertion would require
	// importing mattn/go-sqlite3 here, which forces CGO_ENABLED=1 on
	// every downstream consumer — including PG/MySQL-only services
	// that ship as scratch/distroless. The match is intentionally
	// loose (substring) so a future driver upgrade that adds
	// "(unique constraint)" prefixes or similar still fires.
	//
	// Covered by the sqlite integration test in
	// internal/storages/test/add_task_test.go ("several externalID")
	// — if the driver ever changes the message format the test fails
	// loudly, prompting an update here.
	if msg := err.Error(); strings.Contains(msg, "UNIQUE constraint failed") {
		return entity.ErrDuplicateTask
	}

	return err
}
