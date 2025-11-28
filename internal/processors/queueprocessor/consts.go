package queueprocessor

import "time"

const (
	// Processor constants.
	defaultProcessorMaxAttempts             = 5
	defaultProcessorWorkers                 = 10
	defaultProcessorTimeout                 = 30 * time.Second
	defaultProcessorStaticNextAttemptPeriod = 10 * time.Minute

	// Fetcher constants.
	defaultFetchTick     = 30 * time.Second
	defaultFetchTimeout  = 30 * time.Second
	defaultFetchMaxTasks = int64(100)
)
