package xvalue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Misconfigured arguments degrade to a static default instead of failing:
// no fetch, no polling goroutine (synctest fails if one is left running).
func TestNewDynamicValue_InvalidArgsDegradeToDefault(t *testing.T) {
	t.Parallel()

	fetch := func(context.Context) (int, error) {
		t.Error("fetch must not be called on invalid args")
		return 0, errors.New("unexpected fetch")
	}

	cases := map[string]func(ctx context.Context) *DynamicValue[int]{
		//nolint:staticcheck // nil ctx is the case under test
		"nil ctx":           func(context.Context) *DynamicValue[int] { return NewDynamicValue(nil, 7, fetch, time.Second) },
		"nil fetch":         func(ctx context.Context) *DynamicValue[int] { return NewDynamicValue(ctx, 7, nil, time.Second) },
		"zero interval":     func(ctx context.Context) *DynamicValue[int] { return NewDynamicValue(ctx, 7, fetch, 0) },
		"negative interval": func(ctx context.Context) *DynamicValue[int] { return NewDynamicValue(ctx, 7, fetch, -time.Second) },
	}
	for name, newValue := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx, logs := observedCtx()

				v := newValue(ctx)
				assert.Equal(t, 7, v.Get())

				time.Sleep(5 * time.Second)
				synctest.Wait()
				assert.Equal(t, 7, v.Get())
				if name != "nil ctx" {
					assert.Equal(t, 1, logs.Len(), "misconfiguration is logged")
				}
			})
		})
	}
}

func TestNewDynamicValue_InitialFetch(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			v := NewDynamicValue(ctx, 1, func(context.Context) (int, error) { return 7, nil }, time.Second)
			assert.Equal(t, 7, v.Get())
		})
	})

	t.Run("error falls back to default", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			v := NewDynamicValue(ctx, 5, func(context.Context) (int, error) { return 0, errors.New("boom") }, time.Second)
			assert.Equal(t, 5, v.Get())
		})
	})

	t.Run("fetch receives deadline equal to interval", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			start := time.Now()
			var deadline time.Time
			NewDynamicValue(ctx, 0, func(ctx context.Context) (int, error) {
				deadline, _ = ctx.Deadline()
				return 1, nil
			}, 3*time.Second)
			assert.Equal(t, start.Add(3*time.Second), deadline)
		})
	})
}

func TestDynamicValue_Polling(t *testing.T) {
	t.Parallel()

	t.Run("picks up new value and keeps last good on error", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var (
				mu   sync.Mutex
				next = 1
				fail bool
			)
			fetch := func(context.Context) (int, error) {
				mu.Lock()
				defer mu.Unlock()
				if fail {
					return 0, errors.New("source down")
				}
				return next, nil
			}

			v := NewDynamicValue(ctx, 0, fetch, time.Second)
			assert.Equal(t, 1, v.Get())

			mu.Lock()
			next = 2
			mu.Unlock()
			time.Sleep(time.Second)
			synctest.Wait()
			assert.Equal(t, 2, v.Get())

			mu.Lock()
			fail = true
			mu.Unlock()
			time.Sleep(time.Second)
			synctest.Wait()
			assert.Equal(t, 2, v.Get())
		})
	})

	t.Run("panic in fetch is recovered", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			calls := atomic.Int32{}
			v := NewDynamicValue(ctx, 3, func(context.Context) (int, error) {
				if calls.Add(1) > 1 {
					panic("user bug")
				}
				return 9, nil
			}, time.Second)

			time.Sleep(time.Second)
			synctest.Wait()
			assert.Equal(t, 9, v.Get())
			assert.EqualValues(t, 2, calls.Load())
		})
	})

	t.Run("polling stops after ctx cancel", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())

			calls := atomic.Int32{}
			v := NewDynamicValue(ctx, 0, func(context.Context) (int, error) {
				return int(calls.Add(1)), nil
			}, time.Second)

			time.Sleep(time.Second)
			synctest.Wait()
			require.EqualValues(t, 2, calls.Load())

			cancel()
			synctest.Wait()
			time.Sleep(5 * time.Second)
			synctest.Wait()
			assert.EqualValues(t, 2, calls.Load())
			assert.Equal(t, 2, v.Get())
			// synctest.Test fails if the polling goroutine is still alive here.
		})
	})
}

func TestDynamicValue_ConcurrentGet(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		counter := atomic.Int32{}
		v := NewDynamicValue(ctx, 0, func(context.Context) (int, error) {
			return int(counter.Add(1)), nil
		}, time.Millisecond)

		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 100 {
					_ = v.Get()
					time.Sleep(time.Millisecond)
				}
			})
		}
		wg.Wait()
		assert.Positive(t, v.Get())
	})
}

// TestNewDynamicValue_BlockingFetchBoundedByInterval pins AC7: a fetch that
// honors its ctx but never answers keeps the constructor blocked for at most
// one interval, after which the default is cached.
func TestNewDynamicValue_BlockingFetchBoundedByInterval(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		start := time.Now()
		v := NewDynamicValue(ctx, 5, func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}, 3*time.Second)

		assert.Equal(t, 3*time.Second, time.Since(start))
		assert.Equal(t, 5, v.Get())
	})
}

func TestDynamicValue_EachFailedFetchIsLoggedWithCtx(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		logCtx, logs := observedCtx()
		ctx, cancel := context.WithCancel(logCtx)
		defer cancel()

		NewDynamicValue(ctx, 0, func(context.Context) (int, error) {
			return 0, errors.New("source down")
		}, time.Second)
		assert.Equal(t, 1, logs.Len(), "failed initial fetch is logged")

		time.Sleep(2 * time.Second)
		synctest.Wait()
		assert.Equal(t, 3, logs.Len(), "every failed poll is logged, not deduplicated")
	})
}

func TestDynamicValue_NoWarnOnShutdown(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		logCtx, logs := observedCtx()
		ctx, cancel := context.WithCancel(logCtx)

		NewDynamicValue(ctx, 0, func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}, time.Second)
		require.Equal(t, 1, logs.Len(), "initial fetch timed out")

		time.Sleep(time.Second + time.Millisecond)
		cancel()
		synctest.Wait()
		assert.Equal(t, 1, logs.Len(), "fetch canceled by shutdown is not a failure")
	})
}
