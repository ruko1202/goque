// Package goque provides a distributed task queue manager with support for multiple processors and automatic task healing and cleaning.
package goque

import (
	"context"
	"errors"
	"sync"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

// Goque is the main task queue manager that coordinates multiple task processors.
type Goque struct {
	taskStorage TaskStorage
	processors  map[string]*queueprocessor.GoqueProcessor
}

// NewGoque creates a new Goque instance with the specified task storage.
func NewGoque(taskStorage TaskStorage) *Goque {
	goque := &Goque{
		taskStorage: taskStorage,
		processors:  make(map[string]*queueprocessor.GoqueProcessor),
	}

	return goque
}

// RegisterProcessor registers a new task processor for a specific task type.
// Should be call before Run.
func (g *Goque) RegisterProcessor(
	processorType string,
	taskProcessor queueprocessor.TaskProcessor,
	opts ...queueprocessor.GoqueProcessorOpts,
) {
	g.processors[processorType] = queueprocessor.NewGoqueProcessor(
		g.taskStorage,
		processorType,
		taskProcessor,
		opts...,
	)
}

// Run starts all registered processors in separate goroutines.
func (g *Goque) Run(ctx context.Context) error {
	if len(g.processors) == 0 {
		return errors.New("no processors to run")
	}

	var runErr error
	for _, p := range g.processors {
		err := p.Run(ctx)
		if err != nil {
			xlog.Error(ctx, "failed to run processor", zap.Error(err))
			runErr = errors.Join(runErr, err)
		}
	}

	return runErr
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
