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
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque/internal/metrics"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/processors/internalprocessors"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

type (
	taskFetcher struct {
		taskType entity.TaskType
		maxTasks *xvalue.Positive[int64]
		tick     *xvalue.Positive[time.Duration]
		timeout  *xvalue.Positive[time.Duration]
	}
	taskProcessor struct {
		taskProcessor         TaskProcessor
		workers               *xvalue.Positive[int]
		workerPanicHandler    func(context.Context) func(any)
		timeout               *xvalue.Positive[time.Duration]
		maxAttempts           *xvalue.Positive[int32]
		nextAttemptAtFunc     NextAttemptAtFunc
		hooksBeforeProcessing []HookBeforeProcessing
		hooksAfterProcessing  []HookAfterProcessing
		verboseLogging        bool
	}
)

// GoqueProcessor manages task fetching, processing, and worker pool coordination.
type GoqueProcessor struct {
	globalCtx         context.Context // global context for logging
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
	}

	p.queueHealer = internalprocessors.NewQueueHealer(taskStorage, taskType, func(ctx context.Context) time.Duration {
		return p.processor.timeout.Load(ctx)
	})

	p.fetcher = &taskFetcher{
		taskType: taskType,
		maxTasks: newFetchMaxTasks(xvalue.NewStaticValue(defaultFetchMaxTasks)),
		tick:     newFetchTick(xvalue.NewStaticValue(defaultFetchTick)),
		timeout:  newFetchTimeout(xvalue.NewStaticValue(defaultFetchTimeout)),
	}
	p.processor = &taskProcessor{
		taskProcessor:      processor,
		workers:            newProcessorWorkers(xvalue.NewStaticValue(defaultProcessorWorkers)),
		workerPanicHandler: p.workersPanicHandler,
		timeout:            newProcessorTimeout(xvalue.NewStaticValue(defaultProcessorTimeout)),
		maxAttempts:        newProcessorMaxAttempts(xvalue.NewStaticValue[int32](defaultProcessorMaxAttempts)),
		nextAttemptAtFunc:  StaticNextAttemptAtFunc(defaultProcessorStaticNextAttemptPeriod),
		hooksBeforeProcessing: []HookBeforeProcessing{
			p.metricsBeforeProcessing,
		},
		hooksAfterProcessing: []HookAfterProcessing{
			p.metricsAfterProcessing,
		},
		verboseLogging: true,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.processor.verboseLogging {
		p.processor.hooksBeforeProcessing = append(p.processor.hooksBeforeProcessing, LoggingBeforeProcessing)
		p.processor.hooksAfterProcessing = append(p.processor.hooksAfterProcessing, LoggingAfterProcessing)
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

	workers := p.workersCount(ctx)
	metrics.SetTasksWorkersTotal(p.fetcher.taskType, workers)

	workerPool, err := ants.NewPool(workers,
		ants.WithPanicHandler(p.processor.workerPanicHandler(ctx)),
	)

	if err != nil {
		xlog.Error(ctx, "failed to create pool", xfield.Error(err))
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
	defer close(p.gracefulStoppedCh)

	tick := p.fetcher.tick.Load(ctx)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// ReleaseTimeout releases the pool itself (also when it times out),
			// so no separate Release is needed.
			p.releaseWorkerPool(ctx, workerPool)
			return
		case <-ticker.C:
			tick = p.resetFetchTicker(ctx, ticker, tick)
			p.resizeWorkerPool(ctx, workerPool)

			err := p.fetchAndProcess(ctx, workerPool)
			if err != nil {
				xlog.Error(ctx, "failed to fetch and process tasks", xfield.Error(err))
			}
		}
	}
}

// resetFetchTicker applies a changed fetch tick to the ticker and returns the
// tick in effect. It runs after a tick fires, so the old tick is waited out.
func (p *GoqueProcessor) resetFetchTicker(ctx context.Context, ticker *time.Ticker, current time.Duration) time.Duration {
	tick := p.fetcher.tick.Load(ctx)
	if tick == current {
		return current
	}

	ticker.Reset(tick)
	xlog.Info(ctx, "fetch tick changed",
		xfield.Duration("old", current),
		xfield.Duration("new", tick),
	)

	return tick
}

// workersCount returns the configured workers count capped at maxProcessorWorkers.
func (p *GoqueProcessor) workersCount(ctx context.Context) int {
	workers := p.processor.workers.Load(ctx)
	if workers > maxProcessorWorkers {
		xlog.Warn(ctx, "workers count above the limit, capped",
			xfield.Int("value", workers),
			xfield.Int("applied", maxProcessorWorkers),
		)
		return maxProcessorWorkers
	}

	return workers
}

// resizeWorkerPool applies a changed workers count to the running pool.
// In-flight tasks are not interrupted: on shrink, ants retires excess workers
// once they finish their current task, so Running may exceed Cap for a while,
// and idle workers can still be reused until then. While the pool is full the
// fetch loop is blocked in Submit, so growth applies once a running task finishes.
func (p *GoqueProcessor) resizeWorkerPool(ctx context.Context, workerPool *ants.Pool) {
	workers := p.workersCount(ctx)
	current := workerPool.Cap()
	if workers == current {
		return
	}

	workerPool.Tune(workers)
	metrics.SetTasksWorkersTotal(p.fetcher.taskType, workers)
	xlog.Info(ctx, "workers count changed",
		xfield.Int("old", current),
		xfield.Int("new", workers),
	)
}

// releaseWorkerPool closes the pool and waits for its workers to exit.
//
// The wait is bounded by the processing timeout per busy worker, but never
// shorter than minWorkerPoolReleaseTimeout: ants' ReleaseTimeout also waits
// for the pool's own purge/ticktock goroutines to stop, so even an idle pool
// needs a moment. A timeout here means jobs are still running past the
// deadline and is reported as an error.
func (p *GoqueProcessor) releaseWorkerPool(ctx context.Context, workerPool *ants.Pool) {
	waitJobs := workerPool.Running() + workerPool.Waiting()
	timeout := max(time.Duration(waitJobs)*p.processor.timeout.Load(ctx), minWorkerPoolReleaseTimeout)
	xlog.Info(ctx, "wait jobs before release worker pool",
		xfield.Int("workers count", waitJobs),
		xfield.Duration("timeout", timeout),
	)

	err := workerPool.ReleaseTimeout(timeout)
	if err != nil {
		xlog.Error(ctx, "failed to release workers",
			xfield.Error(err),
			xfield.Int("running", workerPool.Running()),
		)
	}
}

func (p *GoqueProcessor) fetchAndProcess(ctx context.Context, workerPool *ants.Pool) error {
	for _, task := range p.fetchTasks(ctx) {
		// Submit blocks while the pool is full, so a changed workers count is
		// also applied here, not only once per tick.
		p.resizeWorkerPool(ctx, workerPool)
		err := workerPool.Submit(func() {
			ctx := goquectx.WithValues(ctx, task.Metadata)
			ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.fetchAndProcess",
				xfield.String("taskID", task.ID.String()),
			)
			defer span.End()

			select {
			case <-ctx.Done():
				_ = p.returnTaskWhenGracefulShutdown(ctx, task)
				return
			default:
			}

			p.doProcessTask(ctx, task)
		})
		if err != nil {
			xlog.Error(ctx, "failed to submit task",
				xfield.Error(err),
				xfield.String("taskID", task.ID.String()),
			)
			return err
		}
	}
	return nil
}

func (p *GoqueProcessor) fetchTasks(ctx context.Context) []*entity.Task {
	timeout := p.fetcher.timeout.Load(ctx)
	maxTasks := p.fetcher.maxTasks.Load(ctx)
	ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.fetchTasks",
		xfield.Duration("timeout", timeout),
	)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tasks, err := p.taskStorage.GetTasksForProcessing(ctx, p.fetcher.taskType, maxTasks)
	if err != nil {
		metrics.SetOperationsTotal(p.fetcher.taskType, entity.OperationFetch, 0)
		xlog.Error(ctx, "failed to fetch tasks", xfield.Error(err))
		return []*entity.Task{}
	}

	metrics.SetOperationsTotal(p.fetcher.taskType, entity.OperationFetch, len(tasks))

	return tasks
}

func (p *GoqueProcessor) callHooksBefore(ctx context.Context, task *entity.Task) {
	ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.callHooksBefore")
	defer span.End()

	lo.ForEach(p.processor.hooksBeforeProcessing, func(item HookBeforeProcessing, _ int) {
		xlog.AddSpanEvent(ctx, getHookFuncName(item))
		item(ctx, task)
	})
}

func (p *GoqueProcessor) callHooksAfter(ctx context.Context, task *entity.Task, err error) {
	ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.callHooksAfter")
	defer span.End()

	lo.ForEach(p.processor.hooksAfterProcessing, func(item HookAfterProcessing, _ int) {
		xlog.AddSpanEvent(ctx, getHookFuncName(item))
		item(ctx, task, err)
	})
}

// doProcessTask persists the start, runs the task and persists the result.
// Hooks always run in pairs and get every error, including a failed start, in
// which case the task is not processed. A panic is recovered and fails the
// task with ErrTaskPanic like any processing error.
func (p *GoqueProcessor) doProcessTask(ctx context.Context, task *entity.Task) {
	var (
		err     error
		started bool
	)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", entity.ErrTaskPanic, r)
			xlog.Error(ctx, "task processing panicked",
				xfield.Any("panic", r),
				xfield.Binary("stack", debug.Stack()),
			)
		}
		if started {
			if updateErr := p.updateTaskState(ctx, task, err); updateErr != nil {
				err = errors.Join(err, updateErr)
			}
		}
		p.callHooksAfter(ctx, task, err)
	}()

	err = p.updateTaskStateBeforeProcessing(ctx, task)
	started = err == nil

	p.callHooksBefore(ctx, task)
	if started {
		err = p.processTask(ctx, task)
	}
}

func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
	// Cures from processing count as attempts: a task that used them all up
	// is not run again.
	if task.Attempts >= p.processor.maxAttempts.Load(ctx) {
		return entity.ErrTaskAttemptsExhausted
	}

	timeout := p.processor.timeout.Load(ctx)
	ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.processTask",
		xfield.Duration("timeout", timeout),
	)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	promTimer := prometheus.NewTimer(metrics.TaskProcessingDurationSecondsObserver(task.Type, entity.OperationProcessing))
	defer promTimer.ObserveDuration()

	err := p.processor.taskProcessor.ProcessTask(ctx, task)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %s. %w", entity.ErrTaskTimeout, timeout, err)
	case errors.Is(err, entity.ErrTaskReschedule):
		return err
	case err != nil:
		xlog.Error(ctx, "failed to process task", xfield.Error(err))
		return err
	}

	return nil
}

func (p *GoqueProcessor) workersPanicHandler(ctx context.Context) func(any) {
	return func(a any) {
		xlog.Error(ctx, "worker pool panic",
			xfield.Any("err", a),
			xfield.Binary("stack", debug.Stack()),
		)
	}
}
