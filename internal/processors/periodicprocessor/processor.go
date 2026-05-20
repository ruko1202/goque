package periodicprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// TaskQueueManager adds generated periodic job tasks to the queue.
type TaskQueueManager interface {
	AddTaskToQueue(ctx context.Context, task *entity.Task) error
}

// Processor runs one periodic job schedule.
type Processor struct {
	globalCtx        context.Context // global context for logging
	job              *Job
	taskQueueManager TaskQueueManager

	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc
}

// NewProcessor creates a periodic job processor.
func NewProcessor(taskQueueManager TaskQueueManager, job *Job) *Processor {
	return &Processor{
		job:               job,
		taskQueueManager:  taskQueueManager,
		gracefulStoppedCh: make(chan struct{}),
	}
}

// Name returns the processor name.
func (p *Processor) Name() string {
	return fmt.Sprintf("goque-periodic-job-%s", p.job.Name())
}

// Run starts the periodic job processor.
func (p *Processor) Run(ctx context.Context) error {
	ctx = xlog.WithOperation(ctx, p.Name())
	p.globalCtx = ctx

	xlog.Info(ctx, "start periodic job")

	ctx, p.gracefulCtxCancel = context.WithCancel(ctx)
	go p.run(ctx)

	return nil
}

// Stop gracefully shuts down the processor.
func (p *Processor) Stop() {
	xlog.Info(p.globalCtx, "graceful shutdown")
	p.gracefulCtxCancel()
	<-p.gracefulStoppedCh
	xlog.Info(p.globalCtx, "graceful shutdown successful finished")
}

func (p *Processor) run(ctx context.Context) {
	defer close(p.gracefulStoppedCh)

	if p.job.shouldRunOnStart() {
		p.addTaskToQueue(ctx)
	}

	lastRunAt := xtime.Now()
	for {
		nextRunAt := p.job.next(lastRunAt)
		if nextRunAt.IsZero() {
			xlog.Warn(ctx, "no next run: schedule returned zero time")
			return
		}

		if !nextRunAt.After(lastRunAt) {
			xlog.Error(ctx, "periodic job schedule did not advance",
				xfield.Time("last_run_at", lastRunAt),
				xfield.Time("next_run_at", nextRunAt),
			)
			return
		}

		timer := time.NewTimer(time.Until(nextRunAt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			p.addTaskToQueue(ctx)
		}

		lastRunAt = nextRunAt
	}
}

func (p *Processor) addTaskToQueue(ctx context.Context) {
	ctx, span := xlog.WithOperationSpan(ctx, "periodicprocessor.addTaskToQueue")
	defer span.End()

	task, err := p.job.create(ctx)
	if err != nil {
		xlog.Error(ctx, "failed to build periodic job task", xfield.Error(err))
		return
	}
	if task == nil {
		xlog.Error(ctx, "periodic job factory returned nil task")
		return
	}

	if err := p.taskQueueManager.AddTaskToQueue(ctx, task); err != nil {
		xlog.Error(ctx, "failed to add periodic job task to queue", xfield.Error(err))
	}
}
