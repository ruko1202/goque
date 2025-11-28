// Package internalprocessors provides internal task processors for queue management including cleaning and healing operations.
package internalprocessors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ruko1202/goque/internal/entity"
)

const (
	queueCleaner                   = "cleaner"
	defaultCleanerTickPeriod       = 5 * time.Minute
	defaultCleanerTimeout          = 30 * time.Second
	defaultCleanerUpdatedAtTimeAgo = 3 * time.Hour
)

// CleanerTaskStorage defines the storage interface required for the cleaner processor to delete old tasks.
type CleanerTaskStorage interface {
	DeleteTasks(ctx context.Context, taskType entity.TaskType, statuses []entity.TaskStatus, updatedAtTimeAgo time.Duration) ([]*entity.Task, error)
}

// QueueCleaner removes old completed, canceled, or failed tasks from the queue.
type QueueCleaner struct {
	*baseProcessor
	taskStorage CleanerTaskStorage

	taskType         entity.TaskType
	updatedAtTimeAgo time.Duration
}

// NewQueueCleaner creates a new queue cleaner with the specified storage and options.
func NewQueueCleaner(taskStorage CleanerTaskStorage, taskType string) *QueueCleaner {
	q := &QueueCleaner{
		taskStorage:      taskStorage,
		taskType:         taskType,
		updatedAtTimeAgo: defaultCleanerUpdatedAtTimeAgo,
	}
	q.baseProcessor = newBaseProcessor(
		queueCleaner,
		defaultCleanerTimeout,
		defaultCleanerTickPeriod,
		q.CleanTasksQueue,
	)

	return q
}

// SetUpdatedAtTimeAgo sets the time threshold for considering tasks as old enough to be cleaned.
func (q *QueueCleaner) SetUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) {
	q.updatedAtTimeAgo = updatedAtTimeAgo
}

// CleanTasksQueue removes old tasks with done, canceled, or attempts_left status from the queue.
func (q *QueueCleaner) CleanTasksQueue(ctx context.Context) error {
	tasks, err := q.taskStorage.DeleteTasks(ctx, q.taskType, []entity.TaskStatus{
		entity.TaskStatusDone,
		entity.TaskStatusCanceled,
		entity.TaskStatusAttemptsLeft,
	}, q.updatedAtTimeAgo)
	if err != nil {
		slog.ErrorContext(ctx, "failed to clean the queue", slog.Any("err", err))
		return err
	}

	slog.InfoContext(ctx, fmt.Sprintf("cleaned the queue: %d tasks", len(tasks)))
	for _, task := range tasks {
		slog.InfoContext(ctx, "removed task from queue",
			slog.Any("taskID", task.ID),
			slog.String("externalID", task.ExternalID),
			slog.String("type", task.Type),
			slog.String("status", task.Status),
			slog.Any("errors", task.Errors),
			slog.Time("createdAt", task.CreatedAt),
			slog.Any("updatedAt", task.UpdatedAt),
		)
	}

	return nil
}
