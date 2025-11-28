package internalprocessors

import (
	"context"
	"fmt"
	"time"

	"github.com/ruko1202/goque/internal/entity"
)

const (
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

	updatedAtTimeAgo time.Duration
}

// NewQueueCleaner creates a new queue cleaner with the specified storage and options.
func NewQueueCleaner(taskStorage CleanerTaskStorage, taskType entity.TaskType) *QueueCleaner {
	q := &QueueCleaner{
		taskStorage:      taskStorage,
		updatedAtTimeAgo: defaultCleanerUpdatedAtTimeAgo,
	}
	q.baseProcessor = newBaseProcessor(
		entity.OperationCleanup,
		taskType,
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
func (q *QueueCleaner) CleanTasksQueue(ctx context.Context, taskType entity.TaskType) ([]*entity.Task, error) {
	tasks, err := q.taskStorage.DeleteTasks(ctx, taskType, []entity.TaskStatus{
		entity.TaskStatusDone,
		entity.TaskStatusCanceled,
		entity.TaskStatusAttemptsLeft,
	}, q.updatedAtTimeAgo)
	if err != nil {
		return nil, fmt.Errorf("failed to clean the queue: %w", err)
	}

	return tasks, nil
}
