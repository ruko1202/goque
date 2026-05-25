package goque

import (
	"context"

	"github.com/google/uuid"

	"github.com/ruko1202/goque/internal/queuemanager"
)

// TaskQueueManager provides operations for managing tasks in the queue.
// It offers both synchronous and asynchronous methods for adding tasks,
// as well as querying and managing existing tasks.
type TaskQueueManager interface {
	// AsyncAddTaskToQueue enqueues task in a background goroutine
	// and returns immediately. Errors are logged, not returned. If
	// ctx carries a *sqlx.Tx (via WithTx) it is stripped before
	// dispatch — the async goroutine outlives the caller's
	// Commit/Rollback, so enrolling it in the caller's tx would
	// race the close. For outbox semantics use AddTaskToQueue.
	AsyncAddTaskToQueue(ctx context.Context, task *Task)

	// AddTaskToQueue enqueues task synchronously and returns any
	// storage error. Honors a *sqlx.Tx attached to ctx via WithTx
	// (transactional outbox): the insert participates in the
	// caller's tx and is rolled back if the caller rolls back.
	AddTaskToQueue(ctx context.Context, task *Task) error

	// GetTask returns the task with the given ID or an error if it
	// is not found. Honors a tx attached to ctx via WithTx — read
	// goes through the caller's tx if present.
	GetTask(ctx context.Context, taskID uuid.UUID) (*Task, error)

	// GetTasks returns tasks matching filter up to limit. Honors a
	// tx attached to ctx via WithTx.
	GetTasks(ctx context.Context, filter *TaskFilter, limit int64) ([]*Task, error)

	// ResetAttempts clears the retry counter and sets the task back
	// to status=new so it can be picked up again. Runs in its own
	// internal tx and therefore ignores any tx in ctx.
	ResetAttempts(ctx context.Context, taskID uuid.UUID) error

	// CancelTask moves a non-terminal task to status=canceled.
	// No-op if the task is already in a terminal state. Honors a
	// tx attached to ctx via WithTx: both the read and the write
	// participate in the caller's tx, so a rollback unwinds the
	// cancel.
	CancelTask(ctx context.Context, taskID uuid.UUID) error

	// WaitAsyncEnqueues blocks until every in-flight goroutine
	// spawned by AsyncAddTaskToQueue has returned. Called
	// automatically by Goque.Stop(); direct users of
	// TaskQueueManager must call it before closing the underlying
	// *sqlx.DB to avoid "sql: database is closed" errors from late
	// async writes.
	WaitAsyncEnqueues()
}

// NewTaskQueueManager creates a new TaskQueueManager instance with the specified task storage.
func NewTaskQueueManager(taskStorage TaskStorage) TaskQueueManager {
	return queuemanager.NewTaskQueueManager(taskStorage)
}
