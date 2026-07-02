// Package internalprocessors provides internal task processors for queue management including cleaning and healing operations.
package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/metrics"
)

type processQueueFunc func(ctx context.Context, taskType entity.TaskType) ([]*entity.Task, error)

type baseProcessor struct {
	globalCtx         context.Context // global context for logging
	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc

	processorName    string
	processTaskType  entity.TaskType
	processQueueFunc processQueueFunc
	processTimeout   time.Duration
	processPeriod    time.Duration
	processTicker    *time.Ticker

	// startupJitter returns the delay before the first tick. Defaults
	// to defaultStartupJitter (uniform random in [0, period)). Tests
	// override this to make first-tick timing deterministic.
	startupJitter func(period time.Duration) time.Duration
}

func newBaseProcessor(
	processorName string,
	processingTaskType entity.TaskType,
	processTimeout time.Duration,
	processPeriod time.Duration,
	processQueueFunc processQueueFunc,
) *baseProcessor {
	return &baseProcessor{
		processorName:     processorName,
		processTaskType:   processingTaskType,
		processTimeout:    processTimeout,
		processPeriod:     processPeriod,
		processQueueFunc:  processQueueFunc,
		gracefulStoppedCh: make(chan struct{}),
		gracefulCtxCancel: func() {},
		startupJitter:     defaultStartupJitter,
	}
}

func (p *baseProcessor) SetProcessPeriod(processPeriod time.Duration) {
	p.processPeriod = processPeriod
	if p.processTicker != nil {
		p.processTicker.Reset(processPeriod)
	}
}

func (p *baseProcessor) SetProcessTimeout(timeout time.Duration) {
	p.processTimeout = timeout
}

// Run starts the processor, fetching and processing tasks until the context is canceled.
func (p *baseProcessor) Run(ctx context.Context) {
	ctx = xlog.WithOperation(ctx, fmt.Sprintf("internal.processor.%s", p.processorName))
	p.globalCtx = ctx

	xlog.Info(ctx, "start processor")

	ctx, p.gracefulCtxCancel = context.WithCancel(ctx)

	go p.run(ctx)
}

func (p *baseProcessor) run(ctx context.Context) {
	defer close(p.gracefulStoppedCh)

	// Misconfig: a non-positive period would also crash time.NewTicker
	// below. Fail loud at the source instead of starting a useless
	// goroutine that immediately panics.
	if p.processPeriod <= 0 {
		xlog.Error(ctx, "non-positive processPeriod, processor will not start",
			xfield.Duration("processPeriod", p.processPeriod))
		return
	}

	// Stagger the first tick across [0, processPeriod) so 2N
	// concurrently-started cleaners/healers don't all hit the DB
	// in lockstep on every period. See defaultStartupJitter doc.
	jitter := p.startupJitter(p.processPeriod)
	xlog.Info(ctx, "scheduler jitter applied", xfield.Duration("delay", jitter))
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	p.processTicker = time.NewTicker(p.processPeriod)
	defer p.processTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.processTicker.C:
			xlog.Info(ctx, "start processing")
			err := p.doProcessQueue(ctx)
			if err != nil {
				xlog.Error(ctx, "process failed", xfield.Error(err))
			}

			xlog.Info(ctx, "stop processing")
		}
	}
}

// Stop gracefully shuts down the processor by canceling its context.
func (p *baseProcessor) Stop() {
	xlog.Info(p.globalCtx, "graceful shutdown")
	p.gracefulCtxCancel()
	<-p.gracefulStoppedCh
	xlog.Info(p.globalCtx, "graceful shutdown successful finished")
}

func (p *baseProcessor) doProcessQueue(ctx context.Context) error {
	if p.processQueueFunc == nil {
		return errors.New("not implemented")
	}

	xlog.WithFields(ctx,
		xfield.String("action", p.processorName),
		xfield.Duration("timeout", p.processTimeout),
	)

	ctx, cancel := context.WithTimeout(ctx, p.processTimeout)
	defer cancel()

	start := time.Now()
	processedTasks, err := p.processQueueFunc(ctx, p.processTaskType)
	if err != nil {
		xlog.Error(ctx, "process queue failed", xfield.Error(err))
		return fmt.Errorf("process queue failed: %w", err)
	}

	xlog.Infof(ctx, "processed queue: %d tasks", len(processedTasks))

	metrics.SetOperationsTotal(p.processTaskType, p.processorName, len(processedTasks))
	if len(processedTasks) > 0 {
		// Record processing duration only for polls that actually
		// processed at least one task. Internal processors poll on a fixed
		// ticker, so most ticks return zero tasks; observing those empty
		// polls would seed task_processing_duration_seconds with the DB
		// round-trip latency of an empty query and keep histogram_quantile
		// reporting a stale q95 for task types that have no work (e.g.
		// audit.write showing a duration with zero processed tasks).
		// Errored polls are also excluded (the early return above). Note
		// this differs from queueprocessor, which times error paths too;
		// here an empty/failed poll is not "processing" and must not skew
		// the latency distribution.
		metrics.TaskProcessingDurationSecondsObserver(p.processTaskType, p.processorName).
			Observe(time.Since(start).Seconds())
	}

	for _, task := range processedTasks {
		xlog.Info(ctx, "processed queue task",
			xfield.String("taskID", task.ID.String()),
			xfield.String("externalID", task.ExternalID),
			xfield.String("type", task.Type),
			xfield.String("status", task.Status),
			xfield.Any("errors", task.Errors),
			xfield.Time("createdAt", task.CreatedAt),
			xfield.Any("updatedAt", task.UpdatedAt),
		)
	}
	return nil
}

// defaultStartupJitter returns a uniform random delay in [0, period)
// to spread the first tick of a baseProcessor across the period.
// Prevents the thundering-herd at startup: N task-type
// cleaners/healers registered in lockstep would otherwise all hit
// the DB simultaneously every `processPeriod`. After the first
// (jittered) tick the standard ticker takes over and subsequent
// ticks stay spread out for the lifetime of the process.
func defaultStartupJitter(period time.Duration) time.Duration {
	if period <= 0 {
		return 0
	}
	// math/rand/v2 is fine here — this isn't a security primitive,
	// just a uniform spread across the period.
	return rand.N(period) //nolint:gosec
}
