package processor

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/entity"
)

// TaskFetcher defines the interface for fetching tasks for processing.
type TaskFetcher interface {
	FetchTasks(ctx context.Context) ([]*entity.Task, error)
}

// TaskFetcherFunc is a function type that implements the TaskFetcher interface.
type TaskFetcherFunc func(ctx context.Context) ([]*entity.Task, error)

// FetchTasks executes the task fetching function.
func (f TaskFetcherFunc) FetchTasks(ctx context.Context) ([]*entity.Task, error) {
	return f(ctx)
}

func (p *GoqueProcessor) defaultFetchTasks(ctx context.Context) ([]*entity.Task, error) {
	tasks, err := p.taskStorage.GetTasksForProcessing(ctx, p.fetcher.taskType, p.fetcher.maxTasks)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch tasks", slog.Any("err", err))
		return nil, err
	}

	return tasks, nil
}
