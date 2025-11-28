package goque

import (
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

// TaskProcessorFunc is a function type that implements the TaskProcessor interface.
type TaskProcessorFunc = queueprocessor.TaskProcessorFunc

// NoopTaskProcessor is a no-op task processor that does nothing and returns nil.
var NoopTaskProcessor = queueprocessor.NoopTaskProcessor

// Task fetcher configuration options.
var (
	// WithTaskFetcherMaxTasks sets the maximum number of tasks to fetch in a single batch.
	WithTaskFetcherMaxTasks = queueprocessor.WithTaskFetcherMaxTasks
	// WithTaskFetcherTick sets the interval between task fetch attempts.
	WithTaskFetcherTick = queueprocessor.WithTaskFetcherTick
	// WithTaskFetcherTimeout sets the timeout for fetching tasks from storage.
	WithTaskFetcherTimeout = queueprocessor.WithTaskFetcherTimeout
)

// Worker and task processing configuration options.
var (
	// WithWorkersCount sets the number of concurrent workers for processing tasks.
	WithWorkersCount = queueprocessor.WithWorkersCount
	// WithWorkersPanicHandler sets a custom panic handler for worker goroutines.
	WithWorkersPanicHandler = queueprocessor.WithWorkersPanicHandler
	// WithTaskProcessingTimeout sets the timeout for processing a single task.
	WithTaskProcessingTimeout = queueprocessor.WithTaskProcessingTimeout
	// WithTaskProcessingMaxAttempts sets the maximum number of retry attempts for failed tasks.
	WithTaskProcessingMaxAttempts = queueprocessor.WithTaskProcessingMaxAttempts
	// WithTaskProcessingNextAttemptAtFunc sets a custom function to calculate the next retry time.
	WithTaskProcessingNextAttemptAtFunc = queueprocessor.WithTaskProcessingNextAttemptAtFunc
)

// Hook configuration options for task processing.
var (
	// WithHooksBeforeProcessing sets hooks to execute before processing each task.
	WithHooksBeforeProcessing = queueprocessor.WithHooksBeforeProcessing
	// WithHooksAfterProcessing sets hooks to execute after processing each task.
	WithHooksAfterProcessing = queueprocessor.WithHooksAfterProcessing
)

// Cleaner configuration options for removing old tasks.
var (
	// WithCleanerUpdatedAtTimeAgo sets the age threshold for tasks to be cleaned.
	WithCleanerUpdatedAtTimeAgo = queueprocessor.WithCleanerUpdatedAtTimeAgo
	// WithCleanerTimeout sets the timeout for the cleaner operation.
	WithCleanerTimeout = queueprocessor.WithCleanerTimeout
	// WithCleanerPeriod sets the interval between cleaner runs.
	WithCleanerPeriod = queueprocessor.WithCleanerPeriod
)

// Healer configuration options for fixing stuck tasks.
var (
	// WithHealerUpdatedAtTimeAgo sets the age threshold for tasks to be healed.
	WithHealerUpdatedAtTimeAgo = queueprocessor.WithHealerUpdatedAtTimeAgo
	// WithHealerTimeout sets the timeout for the healer operation.
	WithHealerTimeout = queueprocessor.WithHealerTimeout
	// WithHealerPeriod sets the interval between healer runs.
	WithHealerPeriod = queueprocessor.WithHealerPeriod
)
