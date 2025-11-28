package queueprocessor

import (
	"time"
)

// GoqueProcessorOpts is a function type for configuring GoqueProcessor options.
type GoqueProcessorOpts func(*GoqueProcessor)

// WithTaskFetcherMaxTasks sets the maximum number of tasks to fetch in each cycle.
func WithTaskFetcherMaxTasks(maxTasks int64) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.maxTasks = maxTasks
	}
}

// WithTaskFetcherTick sets the interval between task fetching cycles.
func WithTaskFetcherTick(tick time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.tick = tick
	}
}

// WithTaskFetcherTimeout sets the timeout for task fetching operations.
func WithTaskFetcherTimeout(timeout time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.timeout = timeout
	}
}

// WithTaskProcessingTimeout sets the maximum execution time for a single task.
func WithTaskProcessingTimeout(timeout time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.timeout = timeout
	}
}

// WithWorkersCount sets the number of concurrent workers for processing tasks.
func WithWorkersCount(count int) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.workers = count
	}
}

// WithWorkersPanicHandler sets a custom panic handler for worker pool panics.
func WithWorkersPanicHandler(handler func(any)) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.workerPanicHandler = handler
	}
}

// WithTaskProcessingMaxAttempts sets the maximum number of retry attempts for failed tasks.
func WithTaskProcessingMaxAttempts(maxAttempts int32) GoqueProcessorOpts {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return func(p *GoqueProcessor) {
		p.processor.maxAttempts = maxAttempts
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
func WithCleanerUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetUpdatedAtTimeAgo(updatedAtTimeAgo)
	}
}

// WithCleanerTimeout sets the timeout duration for cleaner operations.
func WithCleanerTimeout(timeout time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetProcessTimeout(timeout)
	}
}

// WithCleanerPeriod sets the interval at which the cleaner processor runs.
func WithCleanerPeriod(period time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueCleaner.SetProcessPeriod(period)
	}
}

// WithHealerUpdatedAtTimeAgo sets the time threshold for considering a task as stuck.
func WithHealerUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetUpdatedAtTimeAgo(updatedAtTimeAgo)
	}
}

// WithHealerTimeout sets the timeout duration for healer operations.
func WithHealerTimeout(timeout time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetProcessTimeout(timeout)
	}
}

// WithHealerPeriod sets the interval at which the healer processor runs.
func WithHealerPeriod(period time.Duration) GoqueProcessorOpts {
	return func(q *GoqueProcessor) {
		q.queueHealer.SetProcessPeriod(period)
	}
}
