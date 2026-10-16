package queueprocessor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/metrics"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

type hookCalls struct {
	before   atomic.Int32
	after    atomic.Int32
	afterErr chan error
	afterSt  chan entity.TaskStatus
}

func newHookCalls() *hookCalls {
	return &hookCalls{afterErr: make(chan error, 1), afterSt: make(chan entity.TaskStatus, 1)}
}

func (h *hookCalls) opts() []GoqueProcessorOpts {
	return []GoqueProcessorOpts{
		WithHooksBeforeProcessing(func(context.Context, *entity.Task) { h.before.Add(1) }),
		WithHooksAfterProcessing(func(_ context.Context, task *entity.Task, err error) {
			h.after.Add(1)
			h.afterSt <- task.Status
			h.afterErr <- err
		}),
	}
}

func conflicts(t *testing.T, taskType string, stage metrics.VersionConflictStage) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "goque_task_version_conflicts_total" {
			continue
		}
		for _, m := range family.GetMetric() {
			labels := make(map[string]string)
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["task_type"] == taskType && labels["stage"] == string(stage) {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func TestGoqueProcessor_StartWriteFailure(t *testing.T) {
	t.Parallel()

	for name, startErr := range map[string]error{
		"conflict":    entity.ErrTaskVersionConflict,
		"other error": errors.New("db is down"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			taskType := "type[start failure " + uuid.NewString() + "]"
			task := newTestTask(taskType)

			processed := atomic.Int32{}
			hooks := newHookCalls()
			goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
				TaskProcessorFunc(func(context.Context, *entity.Task) error {
					processed.Add(1)
					return nil
				}),
				append(hooks.opts(), WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)))...,
			)
			defaultFetcherMock(mocks, taskType, []*entity.Task{task})

			started := make(chan struct{})
			mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					close(started)
					return startErr
				})

			require.NoError(t, goqueProc.Run(testCtx(t)))
			recv(t, started)
			// Hooks still run, with the fetched status and the start error.
			assert.Equal(t, entity.TaskStatusPending, recv(t, hooks.afterSt))
			require.ErrorIs(t, recv(t, hooks.afterErr), startErr)
			goqueProc.Stop()

			assert.Zero(t, processed.Load(), "task must not be processed")
			assert.Equal(t, int32(1), hooks.before.Load())
			assert.Equal(t, int32(1), hooks.after.Load())
			wantConflicts := lo.Ternary(errors.Is(startErr, entity.ErrTaskVersionConflict), 1.0, 0.0)
			assert.InDelta(t, wantConflicts, conflicts(t, taskType, metrics.VersionConflictStageStart), 0)
		})
	}
}

func TestGoqueProcessor_AttemptsExhausted(t *testing.T) {
	t.Parallel()
	taskType := "type[exhausted " + uuid.NewString() + "]"
	task := newTestTask(taskType)
	task.Attempts = 2 // e.g. cured twice

	processed := atomic.Int32{}
	hooks := newHookCalls()
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		TaskProcessorFunc(func(context.Context, *entity.Task) error {
			processed.Add(1)
			return nil
		}),
		append(hooks.opts(),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithTaskProcessingMaxAttempts(xvalue.NewStaticValue[int32](2)),
		)...,
	)
	defaultFetcherMock(mocks, taskType, []*entity.Task{task})
	gomock.InOrder(
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				assert.Equal(t, entity.TaskStatusProcessing, task.Status)
				return nil
			}),
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				assert.Equal(t, entity.TaskStatusAttemptsLeft, task.Status)
				assert.Equal(t, int32(2), task.Attempts, "an exhausted task does not use another attempt")
				assert.Contains(t, lo.FromPtr(task.Errors), entity.ErrTaskAttemptsExhausted.Error())
				return nil
			}),
	)

	require.NoError(t, goqueProc.Run(testCtx(t)))
	defer goqueProc.Stop()

	assert.Equal(t, entity.TaskStatusAttemptsLeft, recv(t, hooks.afterSt))
	require.ErrorIs(t, recv(t, hooks.afterErr), entity.ErrTaskAttemptsExhausted)
	assert.Zero(t, processed.Load(), "task must not be processed")
	assert.Equal(t, int32(1), hooks.before.Load())
}

func TestGoqueProcessor_FinishWriteConflict(t *testing.T) {
	t.Parallel()
	taskType := "type[finish conflict " + uuid.NewString() + "]"
	task := newTestTask(taskType)

	hooks := newHookCalls()
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		TaskProcessorFunc(func(context.Context, *entity.Task) error { return nil }),
		append(hooks.opts(), WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)))...,
	)
	defaultFetcherMock(mocks, taskType, []*entity.Task{task})
	gomock.InOrder(
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				task.Version++
				return nil
			}),
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				assert.Equal(t, entity.TaskStatusDone, task.Status)
				assert.Equal(t, int64(1), task.Version, "finish must use the version bumped by start")
				return entity.ErrTaskVersionConflict
			}),
	)

	require.NoError(t, goqueProc.Run(testCtx(t)))
	defer goqueProc.Stop()

	recv(t, hooks.afterSt)
	require.ErrorIs(t, recv(t, hooks.afterErr), entity.ErrTaskVersionConflict)
	assert.InDelta(t, 1.0, conflicts(t, taskType, metrics.VersionConflictStageFinish), 0)
}

func TestGoqueProcessor_ReturnWriteConflict(t *testing.T) {
	t.Parallel()
	taskType := "type[return conflict " + uuid.NewString() + "]"
	task := newTestTask(taskType)

	running := make(chan struct{})
	hooks := newHookCalls()
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
			close(running)
			<-ctx.Done()
			return ctx.Err()
		}),
		append(hooks.opts(), WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)))...,
	)
	defaultFetcherMock(mocks, taskType, []*entity.Task{task})
	gomock.InOrder(
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).Return(nil),
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				assert.Equal(t, entity.TaskStatusNew, task.Status)
				return entity.ErrTaskVersionConflict
			}),
	)

	require.NoError(t, goqueProc.Run(testCtx(t)))
	recv(t, running)
	goqueProc.Stop()

	require.ErrorIs(t, recv(t, hooks.afterErr), entity.ErrTaskVersionConflict)
	assert.InDelta(t, 1.0, conflicts(t, taskType, metrics.VersionConflictStageReturn), 0)
	assert.Zero(t, conflicts(t, taskType, metrics.VersionConflictStageFinish))
}

// The healer must see the processing timeout set by options, which are applied
// after the healer is created.
func TestGoqueProcessor_HealerUsesProcessingTimeout(t *testing.T) {
	t.Parallel()

	const taskType = "type[healer floor]"
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType, NoopTaskProcessor(),
		WithTaskProcessingTimeout(xvalue.NewStaticValue(40*time.Second)),
		WithHealerUpdatedAtTimeAgo(xvalue.NewStaticValue(time.Second)),
	)

	// Configured 1s is raised to the floor 2*40s + 10s for both statuses
	// (not the default 30s timeout: options apply after the healer is created).
	for _, status := range []entity.TaskStatus{entity.TaskStatusProcessing, entity.TaskStatusPending} {
		mocks.taskStorage.EXPECT().
			CureTasks(gomock.Any(), taskType, status, 90*time.Second, gomock.Any()).
			Return(nil, nil)
	}

	_, err := goqueProc.queueHealer.CureTasks(testCtx(t), taskType)
	require.NoError(t, err)
}

// Whatever the processing outcome, a failed finish write is passed to the
// after-hooks, and only a version conflict is counted.
func TestGoqueProcessor_FinishWriteFailure(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		taskErr      error
		wantStatus   entity.TaskStatus
		writeErr     error
		wantConflict float64
	}{
		"cancel under conflict":       {entity.ErrTaskCancel, entity.TaskStatusCanceled, entity.ErrTaskVersionConflict, 1},
		"reschedule under conflict":   {&entity.RescheduleError{After: time.Hour}, entity.TaskStatusNew, entity.ErrTaskVersionConflict, 1},
		"task error under conflict":   {errors.New("boom"), entity.TaskStatusError, entity.ErrTaskVersionConflict, 1},
		"task error, db write failed": {errors.New("boom"), entity.TaskStatusError, errors.New("db is down"), 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			taskType := "type[finish failure " + uuid.NewString() + "]"
			task := newTestTask(taskType)

			hooks := newHookCalls()
			goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
				TaskProcessorFunc(func(context.Context, *entity.Task) error { return tc.taskErr }),
				append(hooks.opts(), WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)))...,
			)
			defaultFetcherMock(mocks, taskType, []*entity.Task{task})
			gomock.InOrder(
				mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
						task.Version++
						return nil
					}),
				mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
						assert.Equal(t, tc.wantStatus, task.Status)
						assert.Equal(t, int64(1), task.Version)
						return tc.writeErr
					}),
			)

			require.NoError(t, goqueProc.Run(testCtx(t)))
			defer goqueProc.Stop()

			recv(t, hooks.afterSt)
			require.ErrorIs(t, recv(t, hooks.afterErr), tc.writeErr)
			assert.InDelta(t, tc.wantConflict, conflicts(t, taskType, metrics.VersionConflictStageFinish), 0)
		})
	}
}

// A task fetched but not started before shutdown is returned to the queue
// with its fetched version and is never processed or passed to hooks.
func TestGoqueProcessor_ReturnBeforeStart(t *testing.T) {
	t.Parallel()
	taskType := "type[return before start " + uuid.NewString() + "]"
	running, queued := newTestTask(taskType), newTestTask(taskType)
	queued.Version = 7

	processed := atomic.Int32{}
	started := make(chan struct{})
	hooks := newHookCalls()
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
			processed.Add(1)
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}),
		append(hooks.opts(),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			// One worker: the second task waits in Submit until the first ends,
			// which is only after shutdown, so it always starts with ctx done.
			WithWorkersCount(xvalue.NewStaticValue(1)),
		)...,
	)
	defaultFetcherMock(mocks, taskType, []*entity.Task{running, queued})
	mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), running.ID, gomock.Any()).Return(nil).Times(2)
	mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), queued.ID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
			assert.Equal(t, entity.TaskStatusNew, task.Status)
			assert.Equal(t, int64(7), task.Version, "return must be fenced on the fetched version")
			return entity.ErrTaskVersionConflict
		})

	require.NoError(t, goqueProc.Run(testCtx(t)))
	recv(t, started)
	goqueProc.Stop()

	assert.Equal(t, int32(1), processed.Load(), "the queued task must not be processed")
	assert.Equal(t, int32(1), hooks.before.Load())
	assert.Equal(t, int32(1), hooks.after.Load())
}

// A panicking processor fails the task like any error: the result is written,
// an attempt is used, hooks get ErrTaskPanic, and the worker keeps going.
func TestGoqueProcessor_ProcessorPanic(t *testing.T) {
	t.Parallel()
	taskType := "type[panic " + uuid.NewString() + "]"
	panicking, next := newTestTask(taskType), newTestTask(taskType)

	hooks := newHookCalls()
	nextProcessed := make(chan struct{})
	goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
		TaskProcessorFunc(func(_ context.Context, task *entity.Task) error {
			if task.ID == panicking.ID {
				panic("boom")
			}
			close(nextProcessed)
			return nil
		}),
		append(hooks.opts(),
			WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			WithWorkersCount(xvalue.NewStaticValue(1)),
		)...,
	)
	defaultFetcherMock(mocks, taskType, []*entity.Task{panicking, next})
	gomock.InOrder(
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), panicking.ID, gomock.Any()).Return(nil),
		mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), panicking.ID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
				assert.Equal(t, entity.TaskStatusError, task.Status)
				assert.Equal(t, int32(1), task.Attempts)
				assert.Contains(t, lo.FromPtr(task.Errors), "boom")
				return nil
			}),
	)
	mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), next.ID, gomock.Any()).Return(nil).Times(2)

	require.NoError(t, goqueProc.Run(testCtx(t)))
	defer goqueProc.Stop()

	assert.Equal(t, entity.TaskStatusError, recv(t, hooks.afterSt))
	require.ErrorIs(t, recv(t, hooks.afterErr), entity.ErrTaskPanic)
	recv(t, nextProcessed)
}
