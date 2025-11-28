// Package internalprocessors provides internal task processors for queue management including cleaning and healing operations.
package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

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
				xlog.Error(ctx, "process failed", zap.Error(err))
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
		zap.String("action", p.processorName),
		zap.Duration("timeout", p.processTimeout),
	)

	ctx, cancel := context.WithTimeout(ctx, p.processTimeout)
	defer cancel()

	promTimer := prometheus.NewTimer(metrics.TaskProcessingDurationSecondsObserver(p.processTaskType, p.processorName))
	defer promTimer.ObserveDuration()

	processedTasks, err := p.processQueueFunc(ctx, p.processTaskType)
	if err != nil {
		xlog.Error(ctx, "process queue failed", zap.Error(err))
		return fmt.Errorf("process queue failed: %w", err)
	}

	metrics.SetOperationsTotal(p.processTaskType, p.processorName, len(processedTasks))

	xlog.Infof(ctx, "processed queue: %d tasks", len(processedTasks))

	for _, task := range processedTasks {
		xlog.Info(ctx, "processed queue task",
			zap.String("taskID", task.ID.String()),
			zap.String("externalID", task.ExternalID),
			zap.String("type", task.Type),
			zap.String("status", task.Status),
			zap.Any("errors", task.Errors),
			zap.Time("createdAt", task.CreatedAt),
			zap.Any("updatedAt", task.UpdatedAt),
		)
	}
	return nil
}
