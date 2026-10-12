package queueprocessor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

// testValue is a Value the test changes directly, without polling.
type testValue[T xvalue.Number] struct {
	v atomic.Int64
}

func newTestValue[T xvalue.Number](v T) *testValue[T] {
	tv := &testValue[T]{}
	tv.Set(v)
	return tv
}

func (v *testValue[T]) Set(value T) { v.v.Store(int64(value)) }
func (v *testValue[T]) Get() T      { return T(v.v.Load()) }

func newTestTask(taskType string) *entity.Task {
	now := xtime.Now()
	return &entity.Task{
		ID:            uuid.New(),
		Type:          taskType,
		ExternalID:    uuid.NewString(),
		Payload:       "payload",
		Status:        entity.TaskStatusPending,
		CreatedAt:     now,
		NextAttemptAt: now,
	}
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
}

// recv fails the test instead of hanging when nothing arrives.
func recv[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a value")
		var zero T
		return zero
	}
}

// waitFor blocks until ch is closed or ctx is done.
func waitFor(ctx context.Context, ch <-chan struct{}) error {
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func workersGauge(t *testing.T, taskType string) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "goque_processors_workers_count" {
			continue
		}
		for _, m := range family.GetMetric() {
			for _, label := range m.GetLabel() {
				if label.GetName() == "task_type" && label.GetValue() == taskType {
					return m.GetGauge().GetValue()
				}
			}
		}
	}
	return -1
}

func TestGoqueProcessor_DynamicFetcher(t *testing.T) {
	t.Parallel()

	t.Run("maxTasks and timeout are read on every fetch", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic fetcher]"

		maxTasks := newTestValue[int64](5)
		timeout := newTestValue(time.Hour)

		type fetchCall struct {
			maxTasks int64
			timeout  time.Duration
		}
		calls := make(chan fetchCall, 1000)

		goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithTaskFetcherMaxTasks(maxTasks),
			WithTaskFetcherTimeout(timeout),
		)
		mocks.taskStorage.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ entity.TaskType, maxTasks int64) ([]*entity.Task, error) {
				deadline, _ := ctx.Deadline()
				calls <- fetchCall{maxTasks: maxTasks, timeout: time.Until(deadline)}
				return nil, nil
			}).
			AnyTimes()

		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		first := recv(t, calls)
		assert.EqualValues(t, 5, first.maxTasks)
		assert.Greater(t, first.timeout, time.Minute)

		maxTasks.Set(7)
		timeout.Set(time.Second)
		require.Eventually(t, func() bool {
			c := recv(t, calls)
			return c.maxTasks == 7 && c.timeout <= time.Second
		}, 2*time.Second, time.Millisecond)

		// Invalid values are ignored: the last valid ones stay applied.
		maxTasks.Set(0)
		timeout.Set(-time.Second)
		for range len(calls) {
			<-calls
		}
		c := recv(t, calls)
		assert.EqualValues(t, 7, c.maxTasks)
		assert.LessOrEqual(t, c.timeout, time.Second)
		assert.Positive(t, c.timeout)
	})

	t.Run("tick change speeds up fetching", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic tick]"

		tick := newTestValue(time.Hour)
		fetches := atomic.Int32{}

		goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
			WithTaskFetcherTick(tick),
		)
		mocks.taskStorage.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
			DoAndReturn(func(context.Context, entity.TaskType, int64) ([]*entity.Task, error) {
				fetches.Add(1)
				return nil, nil
			}).
			AnyTimes()

		tick.Set(200 * time.Millisecond)
		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		require.Eventually(t, func() bool { return fetches.Load() >= 1 }, 2*time.Second, time.Millisecond)
		tick.Set(5 * time.Millisecond)
		require.Eventually(t, func() bool { return fetches.Load() >= 20 }, 2*time.Second, time.Millisecond)
	})
}

func TestGoqueProcessor_DynamicProcessing(t *testing.T) {
	t.Parallel()

	t.Run("processing timeout is read on every task", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic processing timeout]"

		timeout := newTestValue(time.Hour)
		deadlines := make(chan time.Duration, 10)
		firstDone := make(chan struct{})

		goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
			TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
				deadline, _ := ctx.Deadline()
				deadlines <- time.Until(deadline)
				return nil
			}),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithTaskProcessingTimeout(timeout),
		)
		gomock.InOrder(
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return([]*entity.Task{newTestTask(taskType)}, nil),
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				DoAndReturn(func(ctx context.Context, _ entity.TaskType, _ int64) ([]*entity.Task, error) {
					// Change the value only after the first task has read it.
					if err := waitFor(ctx, firstDone); err != nil {
						return nil, err
					}
					timeout.Set(time.Second)
					return []*entity.Task{newTestTask(taskType)}, nil
				}),
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return(nil, nil).AnyTimes(),
		)
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		assert.Greater(t, recv(t, deadlines), time.Minute)
		close(firstDone)
		second := recv(t, deadlines)
		assert.LessOrEqual(t, second, time.Second)
		assert.Greater(t, second, time.Duration(0))
	})

	t.Run("maxAttempts is read on every failed task", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic max attempts]"

		maxAttempts := newTestValue[int32](5)
		statuses := make(chan entity.TaskStatus, 10)
		firstDone := make(chan struct{})

		goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
			TaskProcessorFunc(func(context.Context, *entity.Task) error { return errors.New("fail") }),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithTaskProcessingMaxAttempts(maxAttempts),
		)
		gomock.InOrder(
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return([]*entity.Task{newTestTask(taskType)}, nil),
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				DoAndReturn(func(ctx context.Context, _ entity.TaskType, _ int64) ([]*entity.Task, error) {
					// Change the value only after the first task has read it.
					if err := waitFor(ctx, firstDone); err != nil {
						return nil, err
					}
					maxAttempts.Set(1)
					return []*entity.Task{newTestTask(taskType)}, nil
				}),
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return(nil, nil).AnyTimes(),
		)
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				if task.Status != entity.TaskStatusProcessing {
					statuses <- task.Status
				}
				return nil
			}).AnyTimes()

		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		assert.Equal(t, entity.TaskStatusError, recv(t, statuses))
		close(firstDone)
		assert.Equal(t, entity.TaskStatusAttemptsLeft, recv(t, statuses))
	})
}

func TestGoqueProcessor_DynamicWorkers(t *testing.T) {
	t.Parallel()

	t.Run("invalid initial value falls back to default", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic workers invalid]"

		goqueProc, _ := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
			WithTaskFetcherTick(xvalue.NewStaticValue(time.Hour)),
			WithWorkersCount(xvalue.NewStaticValue(0)),
		)

		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		assert.EqualValues(t, defaultProcessorWorkers, workersGauge(t, taskType))
	})

	t.Run("pool grows before the next submit and shrinks on the next tick", func(t *testing.T) {
		t.Parallel()
		const taskType = "type[dynamic workers]"

		workers := newTestValue(1)
		var (
			mu       sync.Mutex
			running  int
			gates    = make(map[uuid.UUID]chan struct{})
			started  = make(chan uuid.UUID, 10)
			allTasks = []*entity.Task{newTestTask(taskType), newTestTask(taskType), newTestTask(taskType), newTestTask(taskType)}
		)
		for _, task := range allTasks {
			gates[task.ID] = make(chan struct{})
		}
		releaseAll := sync.OnceFunc(func() {
			for _, gate := range gates {
				select {
				case <-gate:
				default:
					close(gate)
				}
			}
		})
		t.Cleanup(releaseAll)

		goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
			TaskProcessorFunc(func(_ context.Context, task *entity.Task) error {
				mu.Lock()
				running++
				mu.Unlock()
				started <- task.ID
				<-gates[task.ID]
				mu.Lock()
				running--
				mu.Unlock()
				return nil
			}),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithWorkersCount(workers),
		)
		gomock.InOrder(
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return(allTasks, nil),
			mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
				Return(nil, nil).AnyTimes(),
		)
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		require.NoError(t, goqueProc.Run(testCtx(t)))
		defer goqueProc.Stop()

		// One worker: the first task runs, the loop is blocked submitting the second.
		first := recv(t, started)
		assert.EqualValues(t, 1, workersGauge(t, taskType))

		// Raise the workers count, then free the only worker: the blocked submit
		// goes through and the pool is resized before the next submit.
		workers.Set(3)
		close(gates[first])

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return running == 3
		}, 2*time.Second, time.Millisecond)
		assert.EqualValues(t, 3, workersGauge(t, taskType))

		// Shrink: in-flight tasks keep running, the pool capacity follows on the next tick.
		workers.Set(1)
		require.Eventually(t, func() bool { return workersGauge(t, taskType) == 1 }, 2*time.Second, time.Millisecond)
		mu.Lock()
		assert.Equal(t, 3, running, "in-flight tasks are not interrupted")
		mu.Unlock()

		releaseAll()
	})
}

// TestGoqueProcessor_InvalidMaxAttemptsFallsBackToDefault pins the documented
// behavior change: maxAttempts <= 0 no longer clamps to 1, the default (5)
// applies instead.
func TestGoqueProcessor_InvalidMaxAttemptsFallsBackToDefault(t *testing.T) {
	t.Parallel()
	const taskType = "type[invalid max attempts]"

	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
		WithTaskProcessingMaxAttempts(xvalue.NewStaticValue[int32](0)),
	)
	mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	task := newTestTask(taskType)
	task.Attempts = defaultProcessorMaxAttempts - 2
	goqueProc.updateTaskState(testCtx(t), task, errors.New("fail"))
	assert.Equal(t, entity.TaskStatusError, task.Status)

	goqueProc.updateTaskState(testCtx(t), task, errors.New("fail"))
	assert.EqualValues(t, defaultProcessorMaxAttempts, task.Attempts)
	assert.Equal(t, entity.TaskStatusAttemptsLeft, task.Status)
}

// TestGoqueProcessor_WorkersResizedOnTick pins the per-tick resize path (no
// tasks, so no Submit): the pool follows the value, the gauge is updated and
// the change is logged as old -> new; an invalid value keeps the last valid one.
func TestGoqueProcessor_WorkersResizedOnTick(t *testing.T) {
	t.Parallel()
	const taskType = "type[dynamic workers tick]"

	core, logs := observer.New(zap.InfoLevel)
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(core)))

	workers := newTestValue(2)
	fetches := make(chan struct{}, 1000)
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
		WithTaskFetcherTick(xvalue.NewStaticValue(5*time.Millisecond)),
		WithWorkersCount(workers),
	)
	mocks.taskStorage.EXPECT().
		GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
		DoAndReturn(func(context.Context, entity.TaskType, int64) ([]*entity.Task, error) {
			select {
			case fetches <- struct{}{}:
			default:
			}
			return nil, nil
		}).
		AnyTimes()

	// Resize runs before the fetch on every tick, so two fetches observed after
	// a change guarantee at least one full resize with the new value.
	waitTicks := func() {
		for range len(fetches) {
			<-fetches
		}
		<-fetches
		<-fetches
	}

	require.NoError(t, goqueProc.Run(ctx))
	defer goqueProc.Stop()

	waitTicks()
	assert.EqualValues(t, 2, workersGauge(t, taskType))

	workers.Set(4)
	waitTicks()
	assert.EqualValues(t, 4, workersGauge(t, taskType))

	changed := logs.FilterMessage("workers count changed").All()
	require.Len(t, changed, 1)
	assert.EqualValues(t, 2, changed[0].ContextMap()["old"])
	assert.EqualValues(t, 4, changed[0].ContextMap()["new"])

	workers.Set(0)
	waitTicks()
	assert.EqualValues(t, 4, workersGauge(t, taskType), "invalid value keeps the last valid one")
	assert.Equal(t, 1, logs.FilterMessage("workers count changed").Len())
	assert.Positive(t, logs.FilterMessage("invalid option value ignored").Len())
}

// TestGoqueProcessor_ReleaseWaitUsesCurrentProcessingTimeout pins that the
// shutdown wait for in-flight jobs is computed from the processing timeout
// value current at Stop, not the one seen when the task started.
func TestGoqueProcessor_ReleaseWaitUsesCurrentProcessingTimeout(t *testing.T) {
	t.Parallel()
	const taskType = "type[dynamic release wait]"

	core, logs := observer.New(zap.InfoLevel)
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(core)))

	timeout := newTestValue(10 * time.Millisecond)
	started := make(chan struct{})
	release := make(chan struct{})

	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		// Ignores ctx on purpose: it finishes only when the test releases it.
		TaskProcessorFunc(func(context.Context, *entity.Task) error {
			close(started)
			<-release
			return nil
		}),
		WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
		WithTaskProcessingTimeout(timeout),
		WithWorkersCount(xvalue.NewStaticValue(1)),
	)
	gomock.InOrder(
		mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
			Return([]*entity.Task{newTestTask(taskType)}, nil),
		mocks.taskStorage.EXPECT().GetTasksForProcessing(gomock.Any(), taskType, gomock.Any()).
			Return(nil, nil).AnyTimes(),
	)
	mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	require.NoError(t, goqueProc.Run(ctx))
	recv(t, started)
	timeout.Set(time.Minute)

	stopped := make(chan struct{})
	go func() {
		goqueProc.Stop()
		close(stopped)
	}()

	require.Eventually(t, func() bool {
		return logs.FilterMessage("wait jobs before release worker pool").Len() == 1
	}, 5*time.Second, time.Millisecond)
	close(release)
	<-stopped

	wait := logs.FilterMessage("wait jobs before release worker pool").All()[0]
	assert.EqualValues(t, time.Minute, wait.ContextMap()["timeout"], "1 running job x current timeout")
	assert.Zero(t, logs.FilterMessage("failed to release workers").Len())
}

func TestGoqueProcessor_WorkersCountCapped(t *testing.T) {
	t.Parallel()

	observedZapCore, logs := observer.New(zap.WarnLevel)
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(observedZapCore)))

	workers := newTestValue(maxProcessorWorkers + 1)
	goqueProc, _ := initGoqueProcessorWithMocks(t, "type[workers cap]", NoopTaskProcessor(),
		WithWorkersCount(workers),
	)
	assert.Equal(t, maxProcessorWorkers, goqueProc.workersCount(ctx))
	assert.Equal(t, 1, logs.FilterMessage("workers count above the limit, capped").Len())

	workers.Set(maxProcessorWorkers)
	assert.Equal(t, maxProcessorWorkers, goqueProc.workersCount(ctx))
	assert.Equal(t, 1, logs.Len(), "values within the limit are not logged")
}
