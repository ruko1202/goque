package xvalue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// testValue is a Value whose content the test controls directly.
type testValue[T Number] struct {
	v atomic.Int64
}

func newTestValue[T Number](v T) *testValue[T] {
	tv := &testValue[T]{}
	tv.Set(v)
	return tv
}

func (v *testValue[T]) Set(value T) { v.v.Store(int64(value)) }
func (v *testValue[T]) Get() T      { return T(v.v.Load()) }

func observedCtx() (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(zap.WarnLevel)
	return xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(core))), logs
}

func TestPositive(t *testing.T) {
	t.Parallel()

	t.Run("valid value is returned", func(t *testing.T) {
		t.Parallel()
		ctx, _ := observedCtx()

		p := NewPositive("workers", NewStaticValue(7), 10)
		assert.Equal(t, 7, p.Load(ctx))
	})

	t.Run("nil value falls back to default", func(t *testing.T) {
		t.Parallel()
		ctx, _ := observedCtx()

		p := NewPositive[time.Duration]("tick", nil, time.Second)
		assert.Equal(t, time.Second, p.Load(ctx))
	})

	t.Run("invalid before any valid uses default", func(t *testing.T) {
		t.Parallel()
		ctx, logs := observedCtx()

		p := NewPositive("maxTasks", NewStaticValue[int64](0), 100)
		assert.EqualValues(t, 100, p.Load(ctx))
		assert.Equal(t, 1, logs.Len())
	})

	t.Run("invalid after valid uses last valid", func(t *testing.T) {
		t.Parallel()
		ctx, _ := observedCtx()

		v := newTestValue[int32](3)
		p := NewPositive[int32]("maxAttempts", v, 5)
		assert.EqualValues(t, 3, p.Load(ctx))

		v.Set(-1)
		assert.EqualValues(t, 3, p.Load(ctx))

		v.Set(8)
		assert.EqualValues(t, 8, p.Load(ctx))
	})

	t.Run("every invalid read is logged", func(t *testing.T) {
		t.Parallel()
		ctx, logs := observedCtx()

		v := newTestValue(0)
		p := NewPositive[int]("workers", v, 10)

		for range 5 {
			p.Load(ctx)
		}
		assert.Equal(t, 5, logs.Len())

		v.Set(4)
		p.Load(ctx)
		assert.Equal(t, 5, logs.Len(), "valid reads are not logged")
	})

	t.Run("concurrent loads", func(t *testing.T) {
		t.Parallel()
		ctx, logs := observedCtx()

		v := newTestValue(0)
		p := NewPositive[int]("workers", v, 10)

		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				for range 100 {
					assert.Equal(t, 10, p.Load(ctx))
				}
			})
		}
		wg.Wait()
		assert.Equal(t, 16*100, logs.Len())
	})
}

type panickingValue struct{}

func (panickingValue) Get() int { panic("broken value") }

func TestPositive_BrokenValues(t *testing.T) {
	t.Parallel()

	t.Run("panicking Get keeps last valid", func(t *testing.T) {
		t.Parallel()
		ctx, logs := observedCtx()

		p := NewPositive[int]("workers", panickingValue{}, 10)
		assert.Equal(t, 10, p.Load(ctx))
		assert.Equal(t, 10, p.Load(ctx))
		assert.Equal(t, 2, logs.Len())
	})

	t.Run("typed nil value keeps default", func(t *testing.T) {
		t.Parallel()
		ctx, _ := observedCtx()

		var dynamic *DynamicValue[int] // e.g. an uninitialized field
		p := NewPositive[int]("workers", dynamic, 10)
		assert.Equal(t, 10, p.Load(ctx))
	})
}

func TestPositive_DurationLoggedHumanReadable(t *testing.T) {
	t.Parallel()
	ctx, logs := observedCtx()

	p := NewPositive("tick", NewStaticValue(-time.Second), 30*time.Second)
	assert.Equal(t, 30*time.Second, p.Load(ctx))

	require.Equal(t, 1, logs.Len())
	fields := logs.All()[0].ContextMap()
	assert.Equal(t, "-1s", fields["value"])
	assert.Equal(t, "30s", fields["applied"])
}
