// Package storages provides interfaces and implementations for task storage backends.
package storages

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

// Task defines the interface for task storage operations.
type Task interface {
	AddTask(ctx context.Context, task *entity.Task) error
	GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error)
	GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error)
	GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, maxTasks int64) ([]*entity.Task, error)
	UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
	DeleteTasks(ctx context.Context, taskType entity.TaskType, statuses []entity.TaskStatus, updatedAtTimeAgo time.Duration) ([]*entity.Task, error)
	CureTasks(ctx context.Context, taskType entity.TaskType, unhealthStatuses []entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error)
	ResetAttempts(ctx context.Context, taskID uuid.UUID) error
}

// AdvancedTaskStorage is used only for tests.
type AdvancedTaskStorage interface {
	Task
	HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
	GetDB() *sqlx.DB
}
