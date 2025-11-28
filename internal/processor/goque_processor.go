// Package processor provides task queue processing functionality with configurable workers and retry logic.
package processor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
)

// TaskStorage defines the interface for task storage operations.
type TaskStorage interface {
	GetTasksForProcessing(ctx context.Context, taskType string, maxTasks int64) ([]*entity.Task, error)
	UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
}

// TaskProcessor defines the interface for processing individual tasks.
type TaskProcessor interface {
	ProcessTask(ctx context.Context, payload string) error
}

type (
	fetcherConfig struct {
		taskType entity.TaskType
		maxTasks int64
		tick     time.Duration
	}
	processorConfig struct {
		workers               int
		taskTimeout           time.Duration
		maxAttempts           int32
		nextAttemptAtFunc     NextAttemptAtFunc
		hooksBeforeProcessing []HookBeforeProcessing
		hooksAfterProcessing  []HookAfterProcessing
	}
)

// GoqueProcessor manages task fetching, processing, and worker pool coordination.
type GoqueProcessor struct {
	gracefulCtxCancel context.CancelFunc

	taskStorage     TaskStorage
	taskProcessor   TaskProcessor
	fetcherConfig   *fetcherConfig
	processorConfig *processorConfig
}

// NewGoqueProcessor creates a new processor instance with the specified configuration.
func NewGoqueProcessor(
	taskRepo TaskStorage,
	taskType entity.TaskType,
	taskProcessor TaskProcessor,
	opts ...GoqueProcessorOpts,
) *GoqueProcessor {
	p := &GoqueProcessor{
		taskStorage:   taskRepo,
		taskProcessor: taskProcessor,
	}

	p.fetcherConfig = &fetcherConfig{
		taskType: taskType,
		maxTasks: defaultFetchMaxTasks,
		tick:     defaultTick,
	}
	p.processorConfig = &processorConfig{
		workers:           defaultWorkers,
		taskTimeout:       defaultTaskTimeout,
		maxAttempts:       defaultMaxAttempts,
		nextAttemptAtFunc: StaticNextAttemptAtFunc(defaultStaticNextAttemptPeriod),
		hooksBeforeProcessing: []HookBeforeProcessing{
			loggingBeforeProcessing,
			p.updateTaskStateBeforeProcessing,
		},
		hooksAfterProcessing: []HookAfterProcessing{
			loggingAfterProcessing,
			p.updateTaskAfterProcessing,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name returns the processor's name based on its task type.
func (p *GoqueProcessor) Name() string {
	return fmt.Sprintf("goque-processor-%s", p.fetcherConfig.taskType)
}

// Run starts the processor, fetching and processing tasks until the context is canceled.
func (p *GoqueProcessor) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.gracefulCtxCancel = cancel

	workerPool, err := ants.NewPool(p.processorConfig.workers)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pool", slog.Any("err", err))
		return err
	}

	go p.runWithWorkerPool(ctx, workerPool)

	return nil
}

func (p *GoqueProcessor) runWithWorkerPool(ctx context.Context, workerPool *ants.Pool) {
	defer workerPool.Release()

	ticker := time.NewTicker(p.fetcherConfig.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			err := workerPool.ReleaseTimeout(time.Duration(workerPool.Running()) * p.processorConfig.taskTimeout)
			if err != nil {
				slog.ErrorContext(ctx, "failed to release workers", slog.Any("err", err))
			}
			return
		case <-ticker.C:
			err := p.fetchAndProcess(ctx, workerPool)
			if err != nil {
				slog.ErrorContext(ctx, "failed to fetch and process tasks", slog.Any("err", err))
			}
		}
	}
}

// Stop gracefully shuts down the processor by canceling its context.
func (p *GoqueProcessor) Stop() {
	p.gracefulCtxCancel()
}

func (p *GoqueProcessor) fetchAndProcess(ctx context.Context, workerPool *ants.Pool) error {
	tasks, err := p.taskStorage.GetTasksForProcessing(ctx, p.fetcherConfig.taskType, p.fetcherConfig.maxTasks)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch tasks", slog.Any("err", err))
		return err
	}
	for _, task := range tasks {
		err := workerPool.Submit(func() {
			lo.ForEach(p.processorConfig.hooksBeforeProcessing, func(item HookBeforeProcessing, _ int) {
				item(ctx, task)
			})

			taskErr := p.processTask(ctx, task)

			lo.ForEach(p.processorConfig.hooksAfterProcessing, func(item HookAfterProcessing, _ int) {
				item(ctx, task, taskErr)
			})
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to submit task", slog.Any("err", err))
			return err
		}
	}
	return nil
}

func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
	ctx, cancel := context.WithTimeout(ctx, p.processorConfig.taskTimeout)
	defer cancel()

	err := p.taskProcessor.ProcessTask(ctx, task.Payload)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %s", ErrTaskTimeout, p.processorConfig.taskTimeout)
	case err != nil:
		return err
	}

	return nil
}

func (p *GoqueProcessor) updateTaskStateBeforeProcessing(ctx context.Context, task *entity.Task) {
	task.Status = entity.TaskStatusProcessing

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task state", slog.Any("err", err))
	}
}

func (p *GoqueProcessor) updateTaskAfterProcessing(ctx context.Context, task *entity.Task, taskErr error) {
	switch {
	case errors.Is(taskErr, ErrTaskCancel):
		task.Status = entity.TaskStatusCanceled
	case errors.Is(taskErr, context.Canceled):
		slog.InfoContext(ctx, "graceful shutdown: return task to queue", slog.String("taskID", task.ID.String()))
		task.Status = entity.TaskStatusNew
	case taskErr != nil:
		task.Attempts = lo.Ternary(task.Attempts == 0, 1, task.Attempts+1)
		task.AddError(taskErr)

		if task.Attempts >= p.processorConfig.maxAttempts {
			task.Status = entity.TaskStatusAttemptsLeft
		} else {
			task.Status = entity.TaskStatusError
			task.NextAttemptAt = p.processorConfig.nextAttemptAtFunc(task.Attempts)
		}
	default:
		task.Status = entity.TaskStatusDone
	}
	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task state", slog.Any("err", err))
	}
}
