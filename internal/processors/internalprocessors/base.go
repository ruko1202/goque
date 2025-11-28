package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

type baseProcessor struct {
	globalCtx         context.Context
	gracefulStoppedCh chan struct{}
	gracefulCtxCancel context.CancelFunc

	name             string
	processQueueFunc func(ctx context.Context) error
	processTimeout   time.Duration
	processPeriod    time.Duration
	processTicker    *time.Ticker
}

func newBaseProcessor(
	name string,
	processTimeout time.Duration,
	processPeriod time.Duration,
	processQueueFunc func(ctx context.Context) error,
) *baseProcessor {
	return &baseProcessor{
		name:              name,
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

// Name returns the processor's name based on its task type.
func (p *baseProcessor) Name() string {
	return fmt.Sprintf("goque-internal-processor-%s", p.name)
}

// Run starts the processor, fetching and processing tasks until the context is canceled.
func (p *baseProcessor) Run(ctx context.Context) {
	ctx = xlog.WithOperation(ctx, p.Name())
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
	ctx, cancel := context.WithTimeout(ctx, p.processTimeout)
	defer cancel()

	if p.processQueueFunc != nil {
		return p.processQueueFunc(ctx)
	}
	return errors.New("not implemented")
}
