package processor

import "time"

const (
	defaultTick                    = 30 * time.Second
	defaultTaskTimeout             = 30 * time.Second
	defaultStaticNextAttemptPeriod = 10 * time.Minute
	defaultWorkers                 = 10
	defaultMaxAttempts             = 5
	defaultFetchMaxTasks           = int64(100)
)
