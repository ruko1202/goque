// Package internalprocessors provides internal task processors for queue management including cleaning and healing operations.
package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/metrics"
)

type processQueueFunc func(ctx context.Context, taskType entity.TaskType) ([]*entity.Task, error)

type baseProcessor struct {
	globalCtx         context.Context
	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc

	processorName    string
	processTaskType  entity.TaskType
	processQueueFunc processQueueFunc
	processTimeout   time.Duration
	processPeriod    time.Duration
	processTicker    *time.Ticker
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
	defer func() {
		p.gracefulStoppedCh <- struct{}{}
	}()

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

	promTimer := prometheus.NewTimer(metrics.TaskProcessingDurationSecondsObserver(p.processTaskType, p.processorName))
	defer promTimer.ObserveDuration()

	processedTasks, err := p.processQueueFunc(ctx, p.processTaskType)
	if err != nil {
		xlog.Error(ctx, "process queue failed", xfield.Error(err))
		return fmt.Errorf("process queue failed: %w", err)
	}

	metrics.SetOperationsTotal(p.processTaskType, p.processorName, len(processedTasks))

	xlog.Infof(ctx, "processed queue: %d tasks", len(processedTasks))

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
