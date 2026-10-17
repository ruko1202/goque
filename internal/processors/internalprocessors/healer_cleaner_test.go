package internalprocessors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/ruko1202/goque/internal/entity"
)

type healerStorageFunc func(updatedAtTimeAgo time.Duration)

func (f healerStorageFunc) CureTasks(_ context.Context, _ entity.TaskType, _ entity.TaskStatus, updatedAtTimeAgo time.Duration, _ string) ([]*entity.Task, error) {
	f(updatedAtTimeAgo)
	return nil, nil
}

type cleanerStorageFunc func(updatedAtTimeAgo time.Duration)

func (f cleanerStorageFunc) DeleteTasks(_ context.Context, _ entity.TaskType, _ []entity.TaskStatus, updatedAtTimeAgo time.Duration) ([]*entity.Task, error) {
	f(updatedAtTimeAgo)
	return nil, nil
}

func TestQueueHealer_DynamicUpdatedAtTimeAgo(t *testing.T) {
	t.Parallel()

	var got time.Duration
	h := NewQueueHealer(healerStorageFunc(func(d time.Duration) { got = d }), "t", noTimeout)

	_, err := h.CureTasks(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, defaultHealerUpdatedAtTimeAgo, got)

	value := newTestDuration(time.Minute)
	h.SetUpdatedAtTimeAgo(value)

	_, err = h.CureTasks(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, time.Minute, got)

	value.Set(2 * time.Minute)
	_, err = h.CureTasks(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, 2*time.Minute, got)

	value.Set(0)
	_, err = h.CureTasks(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, 2*time.Minute, got, "invalid value keeps the last valid one")
}

func TestQueueCleaner_DynamicUpdatedAtTimeAgo(t *testing.T) {
	t.Parallel()

	var got time.Duration
	c := NewQueueCleaner(cleanerStorageFunc(func(d time.Duration) { got = d }), "t")

	_, err := c.CleanTasksQueue(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, defaultCleanerUpdatedAtTimeAgo, got)

	value := newTestDuration(time.Minute)
	c.SetUpdatedAtTimeAgo(value)

	_, err = c.CleanTasksQueue(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, time.Minute, got)

	value.Set(-time.Second)
	_, err = c.CleanTasksQueue(t.Context(), "t")
	require.NoError(t, err)
	require.Equal(t, time.Minute, got, "invalid value keeps the last valid one")
}

type healerStatusStorage struct {
	thresholds map[entity.TaskStatus]time.Duration
	calls      int
}

func (s *healerStatusStorage) CureTasks(_ context.Context, _ entity.TaskType, status entity.TaskStatus, updatedAtTimeAgo time.Duration, _ string) ([]*entity.Task, error) {
	s.calls++
	s.thresholds[status] = updatedAtTimeAgo
	return nil, nil
}

func noTimeout(context.Context) time.Duration { return 0 }

func TestQueueHealer_ProcessingFloor(t *testing.T) {
	t.Parallel()

	observedZapCore, observedLogs := observer.New(zap.WarnLevel)
	ctx := xlog.ContextWithLogger(t.Context(), xlog.NewZapAdapter(zap.New(observedZapCore)))

	storage := &healerStatusStorage{thresholds: map[entity.TaskStatus]time.Duration{}}
	timeout := newTestDuration(30 * time.Second) // floor: 2*30s + 10s = 70s
	h := NewQueueHealer(storage, "t", func(context.Context) time.Duration { return timeout.Get() })
	configured := newTestDuration(time.Minute)
	h.SetUpdatedAtTimeAgo(configured)

	// Configured below the floor: both statuses use the floor.
	_, err := h.CureTasks(ctx, "t")
	require.NoError(t, err)
	require.Equal(t, 2, storage.calls)
	require.Equal(t, 70*time.Second, storage.thresholds[entity.TaskStatusProcessing])
	require.Equal(t, 70*time.Second, storage.thresholds[entity.TaskStatusPending])
	require.Equal(t, 1, observedLogs.Len())

	// Still below the floor: WARN on every run, so a misconfiguration stays visible.
	timeout.Set(35 * time.Second) // floor: 80s
	_, err = h.CureTasks(ctx, "t")
	require.NoError(t, err)
	require.Equal(t, 80*time.Second, storage.thresholds[entity.TaskStatusProcessing])
	require.Equal(t, 2, observedLogs.Len())

	// Configured above the floor: configured value for both, no WARN.
	configured.Set(time.Hour)
	_, err = h.CureTasks(ctx, "t")
	require.NoError(t, err)
	require.Equal(t, time.Hour, storage.thresholds[entity.TaskStatusProcessing])
	require.Equal(t, time.Hour, storage.thresholds[entity.TaskStatusPending])
	require.Equal(t, 2, observedLogs.Len())
}

type failingStatusStorage struct {
	failOn entity.TaskStatus
	err    error
}

func (s *failingStatusStorage) CureTasks(_ context.Context, _ entity.TaskType, status entity.TaskStatus, _ time.Duration, _ string) ([]*entity.Task, error) {
	if status == s.failOn {
		return nil, s.err
	}
	return []*entity.Task{{Status: status}}, nil
}

// The two cure calls are independent: one failing must not
// hide the tasks the other cured, and its error must still be reported.
func TestQueueHealer_ProcessingFloorPartialFailure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		failOn, cured entity.TaskStatus
	}{
		{failOn: entity.TaskStatusProcessing, cured: entity.TaskStatusPending},
		{failOn: entity.TaskStatusPending, cured: entity.TaskStatusProcessing},
	} {
		t.Run(tc.failOn, func(t *testing.T) {
			t.Parallel()

			storageErr := errors.New("db is down")
			h := NewQueueHealer(&failingStatusStorage{failOn: tc.failOn, err: storageErr}, "t", noTimeout)

			tasks, err := h.CureTasks(t.Context(), "t")
			require.ErrorIs(t, err, storageErr)
			require.Len(t, tasks, 1)
			require.Equal(t, tc.cured, tasks[0].Status)
		})
	}
}
