// Package internal provides the core task queue management functionality.
package internal

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ruko1202/goque/internal/processor"
)

// Goque is the main task queue manager that coordinates multiple task processors.
type Goque struct {
	taskStorage processor.TaskStorage
	processors  []*processor.GoqueProcessor
}

// NewGoque creates a new Goque instance with the specified task storage.
func NewGoque(taskStorage processor.TaskStorage) *Goque {
	return &Goque{
		taskStorage: taskStorage,
		processors:  make([]*processor.GoqueProcessor, 0),
	}
}

// RegisterProcessor registers a new task processor for a specific task type.
func (g *Goque) RegisterProcessor(
	taskType string,
	taskProcessor processor.TaskProcessor,
	opts ...processor.GoqueProcessorOpts,
) {
	g.processors = append(g.processors, processor.NewGoqueProcessor(
		g.taskStorage,
		taskType,
		taskProcessor,
		opts...,
	))
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
