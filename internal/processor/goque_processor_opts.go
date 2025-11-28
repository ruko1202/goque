package processor

import "time"

// GoqueProcessorOpts is a function type for configuring GoqueProcessor options.
type GoqueProcessorOpts func(*GoqueProcessor)

// WithFetcherMaxTasks sets the maximum number of tasks to fetch in each cycle.
func WithFetcherMaxTasks(maxTasks int64) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcherConfig.maxTasks = maxTasks
	}
}

// WithFetcherTick sets the interval between task fetching cycles.
func WithFetcherTick(tick time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcherConfig.tick = tick
	}
}

// WithTaskTimeout sets the maximum execution time for a single task.
func WithTaskTimeout(taskTimeout time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.taskTimeout = taskTimeout
	}
}

// WithWorkersCount sets the number of concurrent workers for processing tasks.
func WithWorkersCount(count int) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.workers = count
	}
}

// WithMaxAttempts sets the maximum number of retry attempts for failed tasks.
func WithMaxAttempts(maxAttempts int32) GoqueProcessorOpts {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return func(p *GoqueProcessor) {
		p.processorConfig.maxAttempts = maxAttempts
	}
}

// WithNextAttemptAtFunc sets a custom function to calculate the next retry time.
func WithNextAttemptAtFunc(nextAttemptAt NextAttemptAtFunc) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.nextAttemptAtFunc = nextAttemptAt
	}
}

// WithStaticNextAttemptAtFunc sets a fixed retry delay period.
func WithStaticNextAttemptAtFunc(period time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.nextAttemptAtFunc = StaticNextAttemptAtFunc(period)
	}
}

// WithHookBeforeProcessing adds hooks to run before task processing.
func WithHookBeforeProcessing(hooks ...HookBeforeProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.hooksBeforeProcessing = append(p.processorConfig.hooksBeforeProcessing, hooks...)
	}
}

// WithHookAfterProcessing adds hooks to run after task processing completes.
func WithHookAfterProcessing(hooks ...HookAfterProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processorConfig.hooksAfterProcessing = append(p.processorConfig.hooksAfterProcessing, hooks...)
	}
}
