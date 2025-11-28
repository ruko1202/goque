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
	GetTasksForProcessing(ctx context.Context, taskType entity.TaskType, maxTasks int64) ([]*entity.Task, error)
	UpdateTask(ctx context.Context, taskID uuid.UUID, task *entity.Task) error
}

type (
	taskFetcher struct {
		taskFetcher TaskFetcher
		taskType    entity.TaskType
		maxTasks    int64
		tick        time.Duration
		timeout     time.Duration
	}
	taskProcessor struct {
		taskProcessor         TaskProcessor
		workers               int
		timeout               time.Duration
		maxAttempts           int32
		nextAttemptAtFunc     NextAttemptAtFunc
		hooksBeforeProcessing []HookBeforeProcessing
		hooksAfterProcessing  []HookAfterProcessing
	}
)

// GoqueProcessor manages task fetching, processing, and worker pool coordination.
type GoqueProcessor struct {
	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc

	taskStorage TaskStorage

	fetcher   *taskFetcher
	processor *taskProcessor
}

// NewGoqueProcessor creates a new processor instance with the specified configuration.
func NewGoqueProcessor(
	taskStorage TaskStorage,
	taskType entity.TaskType,
	processor TaskProcessor,
	opts ...GoqueProcessorOpts,
) *GoqueProcessor {
	p := &GoqueProcessor{
		gracefulStoppedCh: make(chan struct{}),
		taskStorage:       taskStorage,
	}

	p.fetcher = &taskFetcher{
		taskFetcher: TaskFetcherFunc(p.defaultFetchTasks),
		taskType:    taskType,
		maxTasks:    defaultFetchMaxTasks,
		tick:        defaultFetchTick,
		timeout:     defaultFetchTimeout,
	}
	p.processor = &taskProcessor{
		taskProcessor:     processor,
		workers:           defaultProcessorWorkers,
		timeout:           defaultProcessorTimeout,
		maxAttempts:       defaultProcessorMaxAttempts,
		nextAttemptAtFunc: StaticNextAttemptAtFunc(defaultProcessorStaticNextAttemptPeriod),
		hooksBeforeProcessing: []HookBeforeProcessing{
			LoggingBeforeProcessing,
			p.updateTaskStateBeforeProcessing,
		},
		hooksAfterProcessing: []HookAfterProcessing{
			LoggingAfterProcessing,
			p.updateTaskAfterProcessing,
		},
	}

	p.Tune(opts)

	return p
}

// Tune reconfigures the processor with new options.
func (p *GoqueProcessor) Tune(opts []GoqueProcessorOpts) {
	for _, opt := range opts {
		opt(p)
	}
}

// Name returns the processor's name based on its task type.
func (p *GoqueProcessor) Name() string {
	return fmt.Sprintf("goque-processor-%s", p.fetcher.taskType)
}

// Run starts the processor, fetching and processing tasks until the context is canceled.
func (p *GoqueProcessor) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "start processor", slog.String("processor", p.Name()))

	ctx, p.gracefulCtxCancel = context.WithCancel(ctx)

	workerPool, err := ants.NewPool(p.processor.workers)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pool", slog.Any("err", err))
		return err
	}

	go p.runWithWorkerPool(ctx, workerPool)

	return nil
}

// Stop gracefully shuts down the processor by canceling its context.
func (p *GoqueProcessor) Stop() {
	slog.Info("start graceful shutdown", slog.String("processor", p.Name()))
	p.gracefulCtxCancel()
	<-p.gracefulStoppedCh
	slog.Info("graceful shutdown successful finished", slog.String("processor", p.Name()))
}

func (p *GoqueProcessor) runWithWorkerPool(ctx context.Context, workerPool *ants.Pool) {
	defer func() { p.gracefulStoppedCh <- struct{}{} }()
	defer workerPool.Release()

	ticker := time.NewTicker(p.fetcher.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			err := workerPool.ReleaseTimeout(time.Duration(workerPool.Running())*p.processor.timeout + time.Millisecond)
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

func (p *GoqueProcessor) fetchAndProcess(ctx context.Context, workerPool *ants.Pool) error {
	for _, task := range p.fetchTasks(ctx) {
		err := workerPool.Submit(func() {
			lo.ForEach(p.processor.hooksBeforeProcessing, func(item HookBeforeProcessing, _ int) {
				item(ctx, task)
			})

			taskErr := p.processTask(ctx, task)

			lo.ForEach(p.processor.hooksAfterProcessing, func(item HookAfterProcessing, _ int) {
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

func (p *GoqueProcessor) fetchTasks(ctx context.Context) []*entity.Task {
	ctx, cancel := context.WithTimeout(ctx, p.fetcher.timeout)
	defer cancel()

	tasks, err := p.fetcher.taskFetcher.FetchTasks(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch tasks", slog.Any("err", err))
		return []*entity.Task{}
	}

	return tasks
}

func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
	ctx, cancel := context.WithTimeout(ctx, p.processor.timeout)
	defer cancel()

	err := p.processor.taskProcessor.ProcessTask(ctx, task)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %s", ErrTaskTimeout, p.processor.timeout)
	case err != nil:
		slog.ErrorContext(ctx, "failed to process task", slog.Any("err", err))
		return err
	}

	return nil
}
