package sqlite

import (
	"context"
	"log/slog"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/jmoiron/sqlx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/utils/xtime"

	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
)

// GetTasksForProcessing retrieves and locks tasks ready for processing, updating their status to pending.
func (s *Storage) GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, limit int64) ([]*entity.Task, error) {
	var tasks []*model.Task
	err := dbutils.DoInTransaction(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error
		tasks, err = s.getTasksForProcessingTx(ctx, tx, taskType, limit)
		if err != nil {
			return err
		}

		return s.batchUpdateTasksStatusTx(ctx, tx, tasks, entity.TaskStatusPending)
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task for processing", slog.Any("err", err))
		return nil, err
	}

	return fromDBModels(tasks)
}

func (s *Storage) getTasksForProcessingTx(ctx context.Context, tx *sqlx.Tx, taskType entity.TaskType, limit int64) ([]*model.Task, error) {
	// SQLite doesn't support FOR UPDATE SKIP LOCKED
	// In WAL mode, the transaction provides row-level locking automatically
	// The forUpdate parameter is kept for interface compatibility but not used
	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(
			sqlite.AND(
				table.Task.Type.EQ(sqlite.String(taskType)),
				table.Task.Status.IN(
					sqlite.String(entity.TaskStatusNew),
					sqlite.String(entity.TaskStatusError),
				),
				table.Task.NextAttemptAt.LT_EQ(sqlite.String(timeToString(xtime.Now()))),
			),
		).
		ORDER_BY(
			table.Task.NextAttemptAt.ASC(),
		).
		LIMIT(limit)

	query, args := stmt.Sql()

	tasks := make([]*model.Task, 0)
	err := tx.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get task for processing", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
