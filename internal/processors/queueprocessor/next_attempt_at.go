package queueprocessor

import (
	"time"

	"github.com/ruko1202/goque/internal/utils/xtime"
)

// NextAttemptAtFunc is a function type that calculates the next retry time based on attempt number.
type NextAttemptAtFunc func(currentAttempt int32) time.Time

// StaticNextAttemptAtFunc creates a function that returns a fixed delay for all retry attempts.
func StaticNextAttemptAtFunc(period time.Duration) NextAttemptAtFunc {
	return func(_ int32) time.Time {
		return xtime.Now().Add(period)
	}
}

// FibonacciPeriods defines retry delays following the Fibonacci sequence.
var FibonacciPeriods = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	3 * time.Minute,
	5 * time.Minute,
	8 * time.Minute,
	13 * time.Minute,
	21 * time.Minute,
	34 * time.Minute,
	55 * time.Minute,
	89 * time.Minute,
}

// StrongPeriods defines a more aggressive retry delay schedule.
var StrongPeriods = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	60 * time.Minute,
}

// StepNextAttemptAtFunc increase next attempt time
//
// example
//
//	attempt: 0, addDuration: 1m0s
//	attempt: 1, addDuration: 5m0s
//	attempt: 2, addDuration: 10m0s
//	attempt: 3, addDuration: 15m0s
//	attempt: 4, addDuration: 30m0s
//	attempt: 5, addDuration: 1h0m0s
//	attempt: 6, addDuration: 1h0m0s
//	attempt: 8, addDuration: 1h0m0s
//	attempt: 9, addDuration: 1h0m0s
func StepNextAttemptAtFunc(periods []time.Duration) NextAttemptAtFunc {
	return func(currentAttempt int32) time.Time {
		addDuration := periods[len(periods)-1]
		if int(currentAttempt) < len(periods) {
			addDuration = periods[currentAttempt]
		}
		return xtime.Now().Add(addDuration)
	}
}

// RoundStepNextAttemptAtFunc round increase next attempt time
//
// example
//
//	attempt: 0, addDuration: 1m0s
//	attempt: 1, addDuration: 5m0s
//	attempt: 2, addDuration: 10m0s
//	attempt: 3, addDuration: 15m0s
//	attempt: 4, addDuration: 30m0s
//	attempt: 5, addDuration: 1h0m0s
//	attempt: 6, addDuration: 1m0s
//	attempt: 7, addDuration: 5m0s
//	attempt: 8, addDuration: 10m0s
//	attempt: 9, addDuration: 15m0s
func RoundStepNextAttemptAtFunc(periods []time.Duration) NextAttemptAtFunc {
	return func(currentAttempt int32) time.Time {
		addDuration := periods[int(currentAttempt)%len(periods)]
		return xtime.Now().Add(addDuration)
	}
}
