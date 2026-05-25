// Package goque provides a distributed task queue manager with support for multiple processors and automatic task healing and cleaning.
package goque

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/utils/xtracer"

	"github.com/ruko1202/goque/internal/processors/periodicprocessor"
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

// Goque is the main task queue manager that coordinates multiple task processors.
type Goque struct {
	taskStorage           TaskStorage
	taskQueueManager      TaskQueueManager
	processors            map[string]*queueprocessor.GoqueProcessor
	periodicJobProcessors map[string]*periodicprocessor.Processor
}

// NewGoque creates a new Goque instance with the specified task storage.
func NewGoque(taskStorage TaskStorage) *Goque {
	return &Goque{
		taskStorage:           taskStorage,
		taskQueueManager:      NewTaskQueueManager(taskStorage),
		processors:            make(map[string]*queueprocessor.GoqueProcessor),
		periodicJobProcessors: make(map[string]*periodicprocessor.Processor),
	}
}

// RegisterProcessor registers a new task processor for a specific task type.
// Should be call before Run.
func (g *Goque) RegisterProcessor(
	processorType string,
	taskProcessor TaskProcessor,
	opts ...ProcessorOpts,
) {
	g.processors[processorType] = queueprocessor.NewGoqueProcessor(
		g.taskStorage,
		processorType,
		taskProcessor,
		opts...,
	)
}

// RegisterPeriodicJob registers a periodic job processor.
// Should be called before Run.
func (g *Goque) RegisterPeriodicJob(job *PeriodicJob) {
	if job == nil {
		return
	}

	g.periodicJobProcessors[job.Name()] = periodicprocessor.NewProcessor(
		g.taskQueueManager,
		job,
	)
}

// Run starts all registered processors in separate goroutines.
func (g *Goque) Run(ctx context.Context) error {
	ctx = xlog.ContextWithTracer(ctx, xtracer.GetTracer())

	if len(g.processors) == 0 && len(g.periodicJobProcessors) == 0 {
		return errors.New("no processors or periodic jobs to run")
	}

	err := g.runProcessors(ctx)
	if err != nil {
		return fmt.Errorf("failed to run processors: %w", err)
	}

	err = g.runPeriodicProcessors(ctx)
	if err != nil {
		return fmt.Errorf("failed to run periodic processors: %w", err)
	}

	return nil
}

func (g *Goque) runProcessors(ctx context.Context) error {
	var runErr error
	for _, p := range g.processors {
		err := p.Run(ctx)
		if err != nil {
			xlog.Error(ctx, "failed to run processor", xfield.Error(err), xfield.String("processor", p.Name()))
			runErr = errors.Join(runErr, fmt.Errorf("failed to run processor '%s': %w", p.Name(), err))
		}
	}

	return runErr
}

func (g *Goque) runPeriodicProcessors(ctx context.Context) error {
	var runErr error

	for _, p := range g.periodicJobProcessors {
		err := p.Run(ctx)
		if err != nil {
			xlog.Error(ctx, "failed to run processor", xfield.Error(err), xfield.String("processor", p.Name()))
			runErr = errors.Join(runErr, fmt.Errorf("failed to run processor '%s': %w", p.Name(), err))
		}
	}

	return runErr
}

// Stop gracefully shuts down all registered processors and waits for them to finish.
//
// Order matters: periodic processors are stopped first (no more new
// tasks dispatched), then queue processors drain in-flight work, then
// any in-flight AsyncAddTaskToQueue goroutines are drained. The last
// step is critical for callers that close the underlying *sqlx.DB
// after Stop() returns — without it a late async write hits a closed
// connection pool.
func (g *Goque) Stop() {
	g.stopPeriodicProcessors()

	g.stopProcessors()

	g.taskQueueManager.WaitAsyncEnqueues()
}

func (g *Goque) stopProcessors() {
	wg := &sync.WaitGroup{}
	for _, p := range g.processors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Stop()
		}()
	}

	wg.Wait()
}

func (g *Goque) stopPeriodicProcessors() {
	wg := &sync.WaitGroup{}
	for _, p := range g.periodicJobProcessors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Stop()
		}()
	}

	wg.Wait()
}
