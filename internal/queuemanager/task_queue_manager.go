// Package queuemanager provides high-level task queue management operations.
// It implements the TaskQueueManager interface and coordinates task storage,
// metrics tracking, and payload validation.
package queuemanager

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"go.opentelemetry.io/otel/trace"

	"github.com/ruko1202/goque/internal/storages/dbtx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/metrics"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/utils/xtracer"
)

const (
	bigPayloadSize = 100 * 1024 // 100KB
)

// TaskQueueManager provides a high-level API for managing tasks in the queue.
// It combines task creation and storage operations in a single interface.
type TaskQueueManager struct {
	taskStorage storages.Task
	tracer      trace.Tracer
}

// NewTaskQueueManager creates a new TaskQueueManager instance with the specified task storage.
func NewTaskQueueManager(taskStorage storages.Task) *TaskQueueManager {
	return &TaskQueueManager{
		taskStorage: taskStorage,
		tracer:      xtracer.GetTracer(),
	}
}

// AsyncAddTaskToQueue adds a task to the queue asynchronously without waiting for completion.
//
// The goroutine outlives the caller's stack, so any *sqlx.Tx carried in ctx
// is stripped before dispatch: enrolling the async write in the caller's tx
// would race against the caller's Commit/Rollback. If a tx is detected the
// write is logged at WARN level and goes against the underlying *sqlx.DB.
func (m *TaskQueueManager) AsyncAddTaskToQueue(ctx context.Context, task *entity.Task) {
	if _, ok := dbtx.TxFromContext(ctx); ok {
		xlog.Warn(ctx, "AsyncAddTaskToQueue called with a tx in context; stripping it to avoid racing the caller's Commit/Rollback",
			xfield.String("task_id", task.ID.String()),
			xfield.String("task_type", task.Type),
		)

		ctx = dbtx.WithoutTx(ctx)
	}
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.AsyncAddTaskToQueue")
	go func() {
		defer span.End()
		err := m.AddTaskToQueue(ctx, task)
		if err != nil {
			xlog.Error(ctx, "failed to async add task to queue", xfield.Error(err))
		}
	}()
}

// AddTaskToQueue adds a task to the queue and returns an error if the operation fails.
func (m *TaskQueueManager) AddTaskToQueue(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.AddTaskToQueue")
	defer span.End()

	metrics.SetTaskPayloadSize(task.Type, len(task.Payload))
	metrics.IncProcessingTasks(task.Type, entity.TaskStatusNew)

	if len(task.Payload) > bigPayloadSize {
		xlog.Warn(ctx, "big payload size detected - may cause performance problems",
			xfield.Int("payload_size", len(task.Payload)),
			xfield.String("task_id", task.ID.String()),
			xfield.String("task_type", task.Type))
	}

	err := m.taskStorage.AddTask(ctx, task)
	if err != nil {
		return err
	}

	return nil
}

// GetTask retrieves a single task by its ID from the queue.
func (m *TaskQueueManager) GetTask(ctx context.Context, taskID uuid.UUID) (*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.GetTask")
	defer span.End()

	return m.taskStorage.GetTask(ctx, taskID)
}

// GetTasks retrieves tasks from the queue based on the provided filter criteria.
// The limit parameter controls the maximum number of tasks to return.
func (m *TaskQueueManager) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.GetTasks")
	defer span.End()

	return m.taskStorage.GetTasks(ctx, filter, limit)
}

// ResetAttempts resets the retry attempts counter for a task and sets its status back to new.
// This allows a failed task to be retried from the beginning.
func (m *TaskQueueManager) ResetAttempts(ctx context.Context, taskID uuid.UUID) error {
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.ResetAttempts")
	defer span.End()

	return m.taskStorage.ResetAttempts(ctx, taskID)
}

// CancelTask marks a non-terminal task as canceled.
func (m *TaskQueueManager) CancelTask(ctx context.Context, taskID uuid.UUID) error {
	ctx, span := xlog.WithOperationSpan(xlog.ContextWithTracer(ctx, m.tracer), "task_queue_manager.CancelTask")
	defer span.End()

	task, err := m.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for cancellation: %w", err)
	}

	if task.IsInTerminalState() {
		return nil
	}

	task.Status = entity.TaskStatusCanceled
	if err := m.taskStorage.UpdateTask(ctx, taskID, task); err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}

	return nil
}
