package internalprocessors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

type healerStorageFunc func(updatedAtTimeAgo time.Duration)

func (f healerStorageFunc) CureTasks(_ context.Context, _ entity.TaskType, _ []entity.TaskStatus, updatedAtTimeAgo time.Duration, _ string) ([]*entity.Task, error) {
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
	h := NewQueueHealer(healerStorageFunc(func(d time.Duration) { got = d }), "t")

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
