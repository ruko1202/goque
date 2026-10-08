package queueprocessor

import "time"

const (
	// Processor constants.
	defaultProcessorMaxAttempts             = 5
	defaultProcessorWorkers                 = 10
	defaultProcessorTimeout                 = 30 * time.Second
	defaultProcessorStaticNextAttemptPeriod = 10 * time.Minute

	// minWorkerPoolReleaseTimeout is the shortest graceful-shutdown wait for
	// the worker pool, so that an idle pool can release cleanly.
	minWorkerPoolReleaseTimeout = time.Second

	// Fetcher constants.
	defaultFetchTick     = 30 * time.Second
	defaultFetchTimeout  = 30 * time.Second
	defaultFetchMaxTasks = int64(100)
)
