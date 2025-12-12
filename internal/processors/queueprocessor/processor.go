// Package queueprocessor provides task queue processing functionality with configurable workers and retry logic.
package queueprocessor

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque/internal/metrics"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/processors/internalprocessors"
	"github.com/ruko1202/goque/internal/storages"
)

type (
	taskFetcher struct {
		taskType entity.TaskType
		maxTasks int64
		tick     time.Duration
		timeout  time.Duration
	}
	taskProcessor struct {
		taskProcessor         TaskProcessor
		workers               int
		workerPanicHandler    func(context.Context) func(any)
		timeout               time.Duration
		maxAttempts           int32
		nextAttemptAtFunc     NextAttemptAtFunc
		hooksBeforeProcessing []HookBeforeProcessing
		hooksAfterProcessing  []HookAfterProcessing
	}
)

// GoqueProcessor manages task fetching, processing, and worker pool coordination.
type GoqueProcessor struct {
	globalCtx         context.Context
	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc

	taskStorage storages.Task

	fetcher      *taskFetcher
	processor    *taskProcessor
	queueCleaner *internalprocessors.QueueCleaner
	queueHealer  *internalprocessors.QueueHealer
}

// NewGoqueProcessor creates a new processor instance with the specified configuration.
func NewGoqueProcessor(
	taskStorage storages.Task,
	taskType entity.TaskType,
	processor TaskProcessor,
	opts ...GoqueProcessorOpts,
) *GoqueProcessor {
	p := &GoqueProcessor{
		gracefulStoppedCh: make(chan struct{}),
		taskStorage:       taskStorage,
		queueCleaner:      internalprocessors.NewQueueCleaner(taskStorage, taskType),
		queueHealer:       internalprocessors.NewQueueHealer(taskStorage, taskType),
	}

	p.fetcher = &taskFetcher{
		taskType: taskType,
		maxTasks: defaultFetchMaxTasks,
		tick:     defaultFetchTick,
		timeout:  defaultFetchTimeout,
	}
	p.processor = &taskProcessor{
		taskProcessor:      processor,
		workers:            defaultProcessorWorkers,
		workerPanicHandler: p.workersPanicHandler,
		timeout:            defaultProcessorTimeout,
		maxAttempts:        defaultProcessorMaxAttempts,
		nextAttemptAtFunc:  StaticNextAttemptAtFunc(defaultProcessorStaticNextAttemptPeriod),
		hooksBeforeProcessing: []HookBeforeProcessing{
			p.updateTaskStateBeforeProcessing,
			p.metricsBeforeProcessing,
			LoggingBeforeProcessing,
		},
		hooksAfterProcessing: []HookAfterProcessing{
			p.updateTaskState,
			p.metricsAfterProcessing,
			LoggingAfterProcessing,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name returns the processor's name based on its task type.
func (p *GoqueProcessor) Name() string {
	return fmt.Sprintf("goque-processor-%s", p.fetcher.taskType)
}

// Run starts the processor, fetching and processing tasks until the context is canceled.
func (p *GoqueProcessor) Run(ctx context.Context) error {
	ctx = xlog.WithOperation(ctx, p.Name())
	p.globalCtx = ctx

	xlog.Info(ctx, "start processor")

	p.queueCleaner.Run(ctx)
	p.queueHealer.Run(ctx)

	ctx, p.gracefulCtxCancel = context.WithCancel(ctx)

	metrics.SetTasksWorkersTotal(p.fetcher.taskType, p.processor.workers)

	workerPool, err := ants.NewPool(p.processor.workers,
		ants.WithPanicHandler(p.processor.workerPanicHandler(ctx)),
	)

	if err != nil {
		xlog.Error(ctx, "failed to create pool", zap.Error(err))
		return err
	}

	go p.runWithWorkerPool(ctx, workerPool)

	return nil
}

// Stop gracefully shuts down the processor by canceling its context.
func (p *GoqueProcessor) Stop() {
	xlog.Info(p.globalCtx, "graceful shutdown")
	p.gracefulCtxCancel()
	<-p.gracefulStoppedCh
	xlog.Info(p.globalCtx, "graceful shutdown successful finished")

	p.queueCleaner.Stop()
	p.queueHealer.Stop()
}

func (p *GoqueProcessor) runWithWorkerPool(ctx context.Context, workerPool *ants.Pool) {
	defer func() { p.gracefulStoppedCh <- struct{}{} }()
	defer workerPool.Release()

	ticker := time.NewTicker(p.fetcher.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			waitJobs := workerPool.Running() + workerPool.Waiting()
			xlog.Info(ctx, "wait jobs before release worker pool", zap.Int("workers count", waitJobs))

			err := workerPool.ReleaseTimeout(time.Duration(waitJobs)*p.processor.timeout + time.Millisecond)
			if err != nil {
				xlog.Error(ctx, "failed to release workers", zap.Error(err))
			}
			return
		case <-ticker.C:
			err := p.fetchAndProcess(ctx, workerPool)
			if err != nil {
				xlog.Error(ctx, "failed to fetch and process tasks", zap.Error(err))
			}
		}
	}
}

func (p *GoqueProcessor) fetchAndProcess(ctx context.Context, workerPool *ants.Pool) error {
	for _, task := range p.fetchTasks(ctx) {
		ctx := xlog.WithFields(ctx, zap.String("taskID", task.ID.String()))

		err := workerPool.Submit(func() {
			select {
			case <-ctx.Done():
				p.returnTaskWhenGracefulShutdown(ctx, task)
				return
			default:
			}

			ctx := goquectx.WithValues(ctx, task.Metadata)

			p.callHooksBefore(ctx, task)

			taskErr := p.processTask(ctx, task)

			p.callHooksAfter(ctx, task, taskErr)
		})
		if err != nil {
			xlog.Error(ctx, "failed to submit task", zap.Error(err))
			return err
		}
	}
	return nil
}

func (p *GoqueProcessor) fetchTasks(ctx context.Context) []*entity.Task {
	ctx = xlog.WithFields(ctx,
		zap.String("processor.action", "fetchTasks"),
		zap.Duration("timeout", p.fetcher.timeout),
	)

	ctx, cancel := context.WithTimeout(ctx, p.fetcher.timeout)
	defer cancel()

	tasks, err := p.taskStorage.GetTasksForProcessing(ctx, p.fetcher.taskType, p.fetcher.maxTasks)
	if err != nil {
		metrics.SetOperationsTotal(p.fetcher.taskType, entity.OperationFetch, 0)
		xlog.Error(ctx, "failed to fetch tasks", zap.Error(err))
		return []*entity.Task{}
	}

	metrics.SetOperationsTotal(p.fetcher.taskType, entity.OperationFetch, len(tasks))

	return tasks
}

func (p *GoqueProcessor) callHooksBefore(ctx context.Context, task *entity.Task) {
	ctx = xlog.WithFields(ctx, zap.String("processor.action", "hooks before"))

	lo.ForEach(p.processor.hooksBeforeProcessing, func(item HookBeforeProcessing, _ int) {
		item(ctx, task)
	})
}

func (p *GoqueProcessor) callHooksAfter(ctx context.Context, task *entity.Task, err error) {
	ctx = xlog.WithFields(ctx, zap.String("processor.action", "hooks after"))

	lo.ForEach(p.processor.hooksAfterProcessing, func(item HookAfterProcessing, _ int) {
		item(ctx, task, err)
	})
}

func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
	ctx = xlog.WithFields(ctx,
		zap.String("processor.action", "processTask"),
		zap.Duration("timeout", p.processor.timeout),
	)

	ctx, cancel := context.WithTimeout(ctx, p.processor.timeout)
	defer cancel()

	promTimer := prometheus.NewTimer(metrics.TaskProcessingDurationSecondsObserver(task.Type, entity.OperationProcessing))
	defer promTimer.ObserveDuration()

	err := p.processor.taskProcessor.ProcessTask(ctx, task)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %s. %w", entity.ErrTaskTimeout, p.processor.timeout, err)
	case err != nil:
		xlog.Error(ctx, "failed to process task", zap.Error(err))
		return err
	}

	return nil
}

func (p *GoqueProcessor) workersPanicHandler(ctx context.Context) func(any) {
	return func(a any) {
		xlog.Error(ctx, "worker pool panic",
			zap.Any("err", a),
			zap.Binary("stack", debug.Stack()),
		)
	}
}
