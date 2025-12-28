package mysqltask

import (
	"context"

	"github.com/go-jet/jet/v2/mysql"
	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/storages/dbutils"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/table"
)

// GetTask retrieves a single task by its ID from the database.
func (s *Storage) GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	task, err := s.getTaskTx(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	return fromDBModel(ctx, task)
}

func (s *Storage) getTaskTx(ctx context.Context, tx dbutils.DBTx, id uuid.UUID) (*model.Task, error) {
	ctx = xlog.WithOperation(ctx, "storage.GetTask",
		zap.String("task_id", id.String()),
	)

	stmt := table.Task.
		SELECT(table.Task.AllColumns).
		WHERE(table.Task.ID.EQ(mysql.String(id.String())))

	dbTask := new(model.Task)
	err := stmt.QueryContext(ctx, tx, dbTask)
	if err != nil {
		xlog.Error(ctx, "failed to get task", zap.Error(err))
		return nil, err
	}

	return dbTask, nil
}
