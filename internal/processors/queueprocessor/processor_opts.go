package queueprocessor

import (
	"context"
	"time"

	"github.com/ruko1202/goque/internal/utils/xvalue"
)

// GoqueProcessorOpts is a function type for configuring GoqueProcessor options.
type GoqueProcessorOpts func(*GoqueProcessor)

// WithTaskFetcherMaxTasks sets the maximum number of tasks to fetch in each cycle.
// The value is re-read on every fetch.
func WithTaskFetcherMaxTasks(maxTasks xvalue.Value[int64]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.maxTasks = newFetchMaxTasks(maxTasks)
	}
}

// WithTaskFetcherTick sets the interval between task fetching cycles.
// A changed value takes effect after the current tick fires.
func WithTaskFetcherTick(tick xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.tick = newFetchTick(tick)
	}
}

// WithTaskFetcherTimeout sets the timeout for task fetching operations.
// The value is re-read on every fetch.
func WithTaskFetcherTimeout(timeout xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.timeout = newFetchTimeout(timeout)
	}
}

// WithTaskProcessingTimeout sets the maximum execution time for a single task.
// The value is re-read for every task.
func WithTaskProcessingTimeout(timeout xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.timeout = newProcessorTimeout(timeout)
	}
}

// WithWorkersCount sets the number of concurrent workers for processing tasks.
// A changed value resizes the running pool on the next tick or before the next
// task submit; in-flight tasks are not interrupted. While the pool is full,
// growth applies once a running task finishes. Values above 10000 are capped.
func WithWorkersCount(count xvalue.Value[int]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.workers = newProcessorWorkers(count)
	}
}

// WithWorkersPanicHandler sets a custom panic handler for worker pool panics.
func WithWorkersPanicHandler(handler func(ctx context.Context) func(any)) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.workerPanicHandler = handler
	}
}

// WithTaskProcessingMaxAttempts sets the maximum number of retry attempts for failed tasks.
// The value is re-read for every failed task.
func WithTaskProcessingMaxAttempts(maxAttempts xvalue.Value[int32]) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.maxAttempts = newProcessorMaxAttempts(maxAttempts)
	}
}

// WithTaskProcessingNextAttemptAtFunc sets a custom function to calculate the next retry time.
func WithTaskProcessingNextAttemptAtFunc(nextAttemptAt NextAttemptAtFunc) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.nextAttemptAtFunc = nextAttemptAt
	}
}

// WithHooksBeforeProcessing adds hooks to run before task processing.
func WithHooksBeforeProcessing(hooks ...HookBeforeProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksBeforeProcessing = append(p.processor.hooksBeforeProcessing, hooks...)
	}
}

// WithHooksAfterProcessing adds hooks to run after task processing completes.
func WithHooksAfterProcessing(hooks ...HookAfterProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksAfterProcessing = append(p.processor.hooksAfterProcessing, hooks...)
	}
}

// WithCleanerUpdatedAtTimeAgo sets the time threshold for considering tasks as old enough to be cleaned.
// The value is re-read on every cleaner run.
func WithCleanerUpdatedAtTimeAgo(updatedAtTimeAgo xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetUpdatedAtTimeAgo(updatedAtTimeAgo)
	}
}

// WithCleanerTimeout sets the timeout duration for cleaner operations.
// The value is re-read on every cleaner run.
func WithCleanerTimeout(timeout xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetProcessTimeout(timeout)
	}
}

// WithCleanerPeriod sets the interval at which the cleaner processor runs.
// A changed value takes effect after the current period elapses.
func WithCleanerPeriod(period xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetProcessPeriod(period)
	}
}

// WithHealerUpdatedAtTimeAgo sets the time threshold for considering a task as stuck.
// The value is re-read on every healer run.
func WithHealerUpdatedAtTimeAgo(updatedAtTimeAgo xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetUpdatedAtTimeAgo(updatedAtTimeAgo)
	}
}

// WithHealerTimeout sets the timeout duration for healer operations.
// The value is re-read on every healer run.
func WithHealerTimeout(timeout xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetProcessTimeout(timeout)
	}
}

// WithHealerPeriod sets the interval at which the healer processor runs.
// A changed value takes effect after the current period elapses.
func WithHealerPeriod(period xvalue.Value[time.Duration]) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetProcessPeriod(period)
	}
}

// WithDisableVerboseLogging disables default verbose logging hooks (LoggingBeforeProcessing, LoggingAfterProcessing).
// Use this option in production to reduce memory allocations from logging.
// Custom logging hooks added via WithHooksBeforeProcessing/WithHooksAfterProcessing will still execute.
func WithDisableVerboseLogging() GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.processor.verboseLogging = false
	}
}

// Validated option values. Each falls back to its default when the configured
// value is missing or not positive.

func newFetchMaxTasks(v xvalue.Value[int64]) *xvalue.Positive[int64] {
	return xvalue.NewPositive("fetcher.max_tasks", v, defaultFetchMaxTasks)
}

func newFetchTick(v xvalue.Value[time.Duration]) *xvalue.Positive[time.Duration] {
	return xvalue.NewPositive("fetcher.tick", v, defaultFetchTick)
}

func newFetchTimeout(v xvalue.Value[time.Duration]) *xvalue.Positive[time.Duration] {
	return xvalue.NewPositive("fetcher.timeout", v, defaultFetchTimeout)
}

func newProcessorWorkers(v xvalue.Value[int]) *xvalue.Positive[int] {
	return xvalue.NewPositive("processor.workers", v, defaultProcessorWorkers)
}

func newProcessorTimeout(v xvalue.Value[time.Duration]) *xvalue.Positive[time.Duration] {
	return xvalue.NewPositive("processor.timeout", v, defaultProcessorTimeout)
}

func newProcessorMaxAttempts(v xvalue.Value[int32]) *xvalue.Positive[int32] {
	return xvalue.NewPositive("processor.max_attempts", v, defaultProcessorMaxAttempts)
}
