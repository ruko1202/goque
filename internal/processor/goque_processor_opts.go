package processor

import (
	"time"

	"github.com/ruko1202/goque/internal/commonopts"
)

const (
	commonProcessorOptsType = "common_opts"
)

// GetGoqueProcessorOpts extracts processor-specific options from a list of internal processor options.
func GetGoqueProcessorOpts(opts []commonopts.InternalProcessorOpt) []GoqueProcessorOpts {
	return commonopts.GetOpts[GoqueProcessorOpts](opts, commonProcessorOptsType)
}

// GoqueProcessorOpts is a function type for configuring GoqueProcessor options.
type GoqueProcessorOpts func(*GoqueProcessor)

// OptionType returns the option type identifier for processor options.
func (o GoqueProcessorOpts) OptionType() string { return commonProcessorOptsType }

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

// WithTaskFetcher replaces the default task fetcher with a custom implementation.
func WithTaskFetcher(fetcher TaskFetcher) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.taskFetcher = fetcher
	}
}

// WithTaskFetcherTimeout sets the timeout for task fetching operations.
func WithTaskFetcherTimeout(timeout time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.fetcher.timeout = timeout
	}
}

// WithTaskTimeout sets the maximum execution time for a single task.
func WithTaskTimeout(timeout time.Duration) GoqueProcessorOpts {
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

// WithMaxAttempts sets the maximum number of retry attempts for failed tasks.
func WithMaxAttempts(maxAttempts int32) GoqueProcessorOpts {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return func(p *GoqueProcessor) {
		p.processor.maxAttempts = maxAttempts
	}
}

// WithNextAttemptAtFunc sets a custom function to calculate the next retry time.
func WithNextAttemptAtFunc(nextAttemptAt NextAttemptAtFunc) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.nextAttemptAtFunc = nextAttemptAt
	}
}

// WithStaticNextAttemptAtFunc sets a fixed retry delay period.
func WithStaticNextAttemptAtFunc(period time.Duration) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.nextAttemptAtFunc = StaticNextAttemptAtFunc(period)
	}
}

// WithHooksBeforeProcessing adds hooks to run before task processing.
func WithHooksBeforeProcessing(hooks ...HookBeforeProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksBeforeProcessing = append(p.processor.hooksBeforeProcessing, hooks...)
	}
}

// WithReplaceHooksBeforeProcessing replaces all hooks with new ones.
func WithReplaceHooksBeforeProcessing(hooks ...HookBeforeProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksBeforeProcessing = hooks
	}
}

// WithHooksAfterProcessing adds hooks to run after task processing completes.
func WithHooksAfterProcessing(hooks ...HookAfterProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksAfterProcessing = append(p.processor.hooksAfterProcessing, hooks...)
	}
}

// WithReplaceHooksAfterProcessing replaces all hooks with new ones.
func WithReplaceHooksAfterProcessing(hooks ...HookAfterProcessing) GoqueProcessorOpts {
	return func(p *GoqueProcessor) {
		p.processor.hooksAfterProcessing = hooks
	}
}
