package xvalue

import (
	"cmp"
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
)

// DynamicValue is a Value backed by a user-provided fetch function that is polled
// in the background. Get always returns the cached value and never performs I/O.
type DynamicValue[T any] struct {
	cache    atomic.Pointer[T]
	fetch    func(ctx context.Context) (T, error)
	interval time.Duration
}

// NewDynamicValue creates a Value that polls fetch every interval and caches the result.
//
// The first fetch is performed synchronously; if it fails, defaultValue is cached.
// When a later fetch fails, the last successful value is kept. Every fetch call
// receives a context derived from ctx with a timeout equal to interval, and fetch
// must honor it: all time bounds assume fetch returns promptly once its ctx is done.
// A panic inside fetch is recovered and treated as a fetch error.
//
// Polling runs in a goroutine that lives until ctx is done; there is no Stop method.
// Using context.Background() keeps that goroutine for the process lifetime.
// Fetch errors are logged using ctx, so attach identifying fields to it via xlog.
//
// Misconfigured arguments (nil ctx, nil fetch, non-positive interval) do not fail:
// a warning is logged and the value stays at defaultValue, like a static one.
func NewDynamicValue[T any](
	ctx context.Context,
	defaultValue T,
	fetch func(ctx context.Context) (T, error),
	interval time.Duration,
) *DynamicValue[T] {
	v := &DynamicValue[T]{
		fetch:    fetch,
		interval: interval,
	}
	v.cache.Store(&defaultValue)

	if ctx == nil || fetch == nil || interval <= 0 {
		xlog.Warn(cmp.Or(ctx, context.Background()), "invalid dynamic value config, using the default value",
			xfield.Bool("nil_ctx", ctx == nil),
			xfield.Bool("nil_fetch", fetch == nil),
			xfield.Duration("interval", interval),
		)
		return v
	}

	v.refresh(ctx)
	go v.poll(ctx)

	return v
}

// Get returns the last successfully fetched value (or the default).
func (v *DynamicValue[T]) Get() T {
	return *v.cache.Load()
}

func (v *DynamicValue[T]) poll(ctx context.Context) {
	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v.refresh(ctx)
		}
	}
}

func (v *DynamicValue[T]) refresh(ctx context.Context) {
	value, err := v.safeFetch(ctx)
	if ctx.Err() != nil {
		// Shutting down: a fetch cut short by cancellation is not a source failure.
		return
	}
	if err != nil {
		xlog.Warn(ctx, "failed to fetch dynamic value, keeping the last one", xfield.Error(err))
		return
	}
	v.cache.Store(&value)
}

func (v *DynamicValue[T]) safeFetch(ctx context.Context) (value T, err error) {
	ctx, cancel := context.WithTimeout(ctx, v.interval)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("dynamic value: fetch panicked: %v", r)
		}
	}()

	return v.fetch(ctx)
}
