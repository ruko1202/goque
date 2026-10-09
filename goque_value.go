package goque

import (
	"context"
	"time"

	"github.com/ruko1202/goque/internal/utils/xvalue"
)

// Value provides a processor configuration value that may change over time.
// Processors read it on every cycle, so implementations must be non-blocking,
// must not panic and must be safe for concurrent use.
type Value[T any] = xvalue.Value[T]

// StaticValue is a Value that never changes.
type StaticValue[T any] = xvalue.StaticValue[T]

// DynamicValue is a Value backed by a polled fetch function, see NewDynamicValue.
type DynamicValue[T any] = xvalue.DynamicValue[T]

// NewStaticValue creates a Value that always returns v.
//
// Go infers T from v, so int64 and int32 options need an explicit type argument,
// e.g. WithTaskFetcherMaxTasks(goque.NewStaticValue[int64](100)).
func NewStaticValue[T any](v T) *StaticValue[T] {
	return xvalue.NewStaticValue(v)
}

// NewDynamicValue creates a Value that polls fetch every interval and caches the result.
//
// The first fetch is performed synchronously; if it fails, defaultValue is cached.
// When a later fetch fails, the last successful value is kept and the error is logged.
// Every fetch call receives a context derived from ctx with a timeout equal to interval,
// and fetch must honor it. A panic inside fetch is recovered and treated as an error.
//
// Polling stops when ctx is done; there is no Stop method, so passing
// context.Background() keeps the polling goroutine for the process lifetime.
// A nil ctx or fetch, or a non-positive interval, is logged and the value then
// stays at defaultValue, like a static one.
func NewDynamicValue[T any](
	ctx context.Context,
	defaultValue T,
	fetch func(ctx context.Context) (T, error),
	interval time.Duration,
) *DynamicValue[T] {
	return xvalue.NewDynamicValue(ctx, defaultValue, fetch, interval)
}
