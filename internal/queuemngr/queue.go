// Package queuemngr provides queue management functionality for adding tasks to storage.
package queuemngr

import (
	"context"

	"github.com/ruko1202/goque/internal/entity"
)

// TaskStorage defines the interface for storing tasks in the queue.
type TaskStorage interface {
	AddTask(ctx context.Context, task *entity.Task) error
}

// QueueMngr manages adding tasks to the queue storage.
type QueueMngr struct {
	taskStorage TaskStorage
}

// NewQueueMngr creates a new QueueMngr instance with the specified task storage.
func NewQueueMngr(taskStorage TaskStorage) *QueueMngr {
	return &QueueMngr{
		taskStorage: taskStorage,
	}
}
