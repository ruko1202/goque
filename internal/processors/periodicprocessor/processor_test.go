package periodicprocessor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mocks/mock_periodicprocessor"
)

func TestProcessor_Run(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	task := entity.NewTask("periodic-test", `{"ok":true}`)

	taskFactory := func(ctx context.Context) (*entity.Task, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		return task, nil
	}

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		calls := atomic.Int32{}
		job, err := NewJob(
			t.Name(),
			SchedulerFunc(func(lastRunAt time.Time) time.Time {
				defer calls.Add(1)

				if calls.Load() <= 1 {
					return lastRunAt.Add(time.Millisecond)
				}

				return lastRunAt
			}),
			taskFactory,
		)
		require.NoError(t, err)

		manager := mock_periodicprocessor.NewMockTaskQueueManager(gomock.NewController(t))
		manager.EXPECT().
			AddTaskToQueue(gomock.Any(), task).
			Return(nil).
			Times(2)

		processor := NewProcessor(manager, job)

		err = processor.Run(ctx)
		require.NoError(t, err)
		<-time.After(500 * time.Millisecond)
		processor.Stop()
	})
	t.Run("run_on_start", func(t *testing.T) {
		t.Parallel()

		job, err := NewJob(
			t.Name(),
			SchedulerFunc(func(t time.Time) time.Time {
				return t.Add(time.Hour)
			}),
			taskFactory,
			WithRunOnStart(),
		)
		require.NoError(t, err)

		manager := mock_periodicprocessor.NewMockTaskQueueManager(gomock.NewController(t))
		manager.EXPECT().
			AddTaskToQueue(gomock.Any(), task).
			Return(nil)

		processor := NewProcessor(manager, job)

		err = processor.Run(ctx)
		require.NoError(t, err)
		<-time.After(500 * time.Millisecond)
		processor.Stop()
	})

	t.Run("schedule does not advance", func(t *testing.T) {
		t.Parallel()

		job, err := NewJob(
			t.Name(),
			SchedulerFunc(func(t time.Time) time.Time {
				return t
			}),
			taskFactory,
		)
		require.NoError(t, err)

		manager := mock_periodicprocessor.NewMockTaskQueueManager(gomock.NewController(t))
		processor := NewProcessor(manager, job)

		err = processor.Run(ctx)
		require.NoError(t, err)
		<-time.After(500 * time.Millisecond)
		processor.Stop()
	})
}
