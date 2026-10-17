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
//
// Writes are fenced by entity.Task.Version, as documented on each method.
type Task interface {
	// AddTask inserts a new task with version 0, whatever task.Version holds.
	AddTask(ctx context.Context, task *entity.Task) error
	GetTask(ctx context.Context, id uuid.UUID) (*entity.Task, error)
	GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error)
	// GetTasksForProcessing moves due tasks to pending and bumps their version;
	// the returned tasks carry the stored version.
	GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, maxTasks int64) ([]*entity.Task, error)
	// UpdateTask writes the task only if the stored version equals task.Version,
	// bumps the stored version and increments task.Version on success. If no row
	// matched (the task changed or was deleted) it returns entity.ErrTaskVersionConflict.
	UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
	DeleteTasks(ctx context.Context, taskType entity.TaskType, statuses []entity.TaskStatus, updatedAtTimeAgo time.Duration) ([]*entity.Task, error)
	// CureTasks moves tasks stuck in status to error and bumps their version; a
	// task cured from processing also consumes an attempt and gets the comment
	// appended to its errors, a task cured from pending keeps both. The returned
	// tasks carry the post-update state.
	CureTasks(ctx context.Context, taskType entity.TaskType, status entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error)
	// ResetAttempts resets the task to new with zero attempts, fenced on the
	// version it read; returns entity.ErrTaskVersionConflict on a concurrent change.
	ResetAttempts(ctx context.Context, taskID uuid.UUID) error
}

// AdvancedTaskStorage is used only for tests.
type AdvancedTaskStorage interface {
	Task
	// HardUpdateTask writes the task unconditionally, without touching updated_at
	// or the version. Test back door only.
	HardUpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
	GetDB() *sqlx.DB
}
