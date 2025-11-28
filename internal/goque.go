// Package internal provides the core task queue management functionality.
package internal

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ruko1202/goque/internal/commonopts"
	internalprocessors "github.com/ruko1202/goque/internal/internal_processors"
	"github.com/ruko1202/goque/internal/processor"
	"github.com/ruko1202/goque/internal/storages/task"
)

// Goque is the main task queue manager that coordinates multiple task processors.
type Goque struct {
	taskStorage        *task.Storage
	processors         map[string]*processor.GoqueProcessor
	internalProcessors map[string]internalprocessors.Processor
}

// NewGoque creates a new Goque instance with the specified task storage.
func NewGoque(taskStorage *task.Storage) *Goque {
	goque := &Goque{
		taskStorage:        taskStorage,
		processors:         make(map[string]*processor.GoqueProcessor),
		internalProcessors: make(map[string]internalprocessors.Processor),
	}
	goque.registerInternalProcessors()

	return goque
}

// RegisterProcessor registers a new task processor for a specific task type.
func (g *Goque) RegisterProcessor(
	processorType string,
	taskProcessor processor.TaskProcessor,
	opts ...processor.GoqueProcessorOpts,
) {
	g.processors[processorType] = processor.NewGoqueProcessor(
		g.taskStorage,
		processorType,
		taskProcessor,
		opts...,
	)
}

// Run starts all registered processors in separate goroutines.
func (g *Goque) Run(ctx context.Context) {
	for _, p := range g.processors {
		go func() {
			err := p.Run(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "failed to run processor", slog.Any("err", err))
			}
		}()
	}
}

// Stop gracefully shuts down all registered processors and waits for them to finish.
func (g *Goque) Stop() {
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

func (g *Goque) registerInternalProcessors() {
	commonProcessorsOpts := []processor.GoqueProcessorOpts{
		processor.WithTaskFetcherTick(5 * time.Minute),
		processor.WithReplaceHooksBeforeProcessing(processor.LoggingBeforeProcessing),
		processor.WithReplaceHooksAfterProcessing(processor.LoggingAfterProcessing),
	}

	// Register healer processor
	healer := internalprocessors.NewQueueHealer(g.taskStorage)

	g.internalProcessors[internalprocessors.Healer] = healer
	g.RegisterProcessor(
		internalprocessors.Healer,
		processor.NoopTaskProcessor(),
		append(commonProcessorsOpts,
			processor.WithTaskFetcherTimeout(internalprocessors.DefaultHealerTimeout),
			processor.WithTaskFetcher(processor.TaskFetcherFunc(healer.CureTasks)),
		)...,
	)

	// Register cleaner processor
	cleaner := internalprocessors.NewQueueCleaner(g.taskStorage)
	g.internalProcessors[internalprocessors.CleanerProcessorName] = cleaner
	g.RegisterProcessor(
		internalprocessors.CleanerProcessorName,
		processor.NoopTaskProcessor(),
		append(commonProcessorsOpts,
			processor.WithTaskFetcherTimeout(internalprocessors.DefaultCleanerTimeout),
			processor.WithTaskFetcher(processor.TaskFetcherFunc(cleaner.CleanTasksQueue)),
		)...,
	)
}

// TuneHealerProcessor reconfigures the healer processor with new options.
func (g *Goque) TuneHealerProcessor(opts ...commonopts.InternalProcessorOpt) {
	g.tuneInternalProcessor(internalprocessors.Healer, opts)
}

// TuneCleanerProcessor reconfigures the cleaner processor with new options.
func (g *Goque) TuneCleanerProcessor(opts ...commonopts.InternalProcessorOpt) {
	g.tuneInternalProcessor(internalprocessors.CleanerProcessorName, opts)
}

func (g *Goque) tuneInternalProcessor(processorType string, opts []commonopts.InternalProcessorOpt) {
	if internalProc, ok := g.internalProcessors[processorType]; ok {
		internalProc.Tune(opts)
	}
	if proc, ok := g.processors[processorType]; ok {
		proc.Tune(processor.GetGoqueProcessorOpts(opts))
	}
}
