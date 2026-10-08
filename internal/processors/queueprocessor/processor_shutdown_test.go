package queueprocessor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

const releaseWorkersErrMsg = "failed to release workers"

// TestGoqueProcessor_StopIdle is a regression test: stopping a processor
// with nothing in flight used to give the worker pool only 1ms to release.
// ants' ReleaseTimeout also waits for the pool's purge/ticktock goroutines
// and spins while doing so, so on a single P (a 1-CPU container) those
// goroutines cannot run before the 1ms timer fires and every idle
// shutdown logged "failed to release workers: operation timed out".
//
// Not parallel on purpose: it pins GOMAXPROCS to 1 to reproduce that
// deterministically, and top-level parallel tests are paused meanwhile.
func TestGoqueProcessor_StopIdle(t *testing.T) {
	prev := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(observedZapCore)))

	goqueProc, _ := initGoqueProcessorWithMocks(t,
		"type[stop idle]",
		TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error { return nil }),
		WithTaskFetcherTick(time.Hour),
	)

	require.NoError(t, goqueProc.Run(ctx))
	require.True(t, stopWithin(goqueProc, 5*time.Second), "processor did not stop in time")

	for _, entry := range observedLogs.FilterLevelExact(zapcore.ErrorLevel).All() {
		t.Errorf("unexpected error log: %q %v", entry.Message, entry.ContextMap())
	}
}

// TestGoqueProcessor_StopReleaseTimeout checks that a genuine release
// timeout (a job still running past the deadline) is still reported.
func TestGoqueProcessor_StopReleaseTimeout(t *testing.T) {
	t.Parallel()

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.New(observedZapCore)))

	now := xtime.Now()
	task := &entity.Task{
		ID:            uuid.New(),
		Type:          "type[stop release timeout]",
		ExternalID:    uuid.NewString(),
		Payload:       "test payload",
		Status:        entity.TaskStatusPending,
		CreatedAt:     now,
		NextAttemptAt: now,
	}

	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	goqueProc, mocks := initGoqueProcessorWithMocks(t,
		task.Type,
		// Ignores context cancellation on purpose, to outlive the release deadline.
		TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error {
			close(started)
			<-release
			return nil
		}),
		WithTaskFetcherTick(50*time.Millisecond),
		WithTaskProcessingTimeout(10*time.Millisecond),
		WithWorkersCount(1),
	)
	defaultFetcherMock(mocks, task.Type, []*entity.Task{task})
	mocks.taskStorage.EXPECT().
		UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	require.NoError(t, goqueProc.Run(ctx))
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("task was not picked up")
	}

	require.True(t, stopWithin(goqueProc, 5*time.Second), "processor did not stop in time")
	require.Equal(t, 1, observedLogs.FilterMessage(releaseWorkersErrMsg).Len())
}

func stopWithin(goqueProc *GoqueProcessor, limit time.Duration) bool {
	done := make(chan struct{})
	go func() {
		goqueProc.Stop()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(limit):
		return false
	}
}
