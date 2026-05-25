package periodicprocessor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mocks/mock_periodicprocessor"
	"github.com/ruko1202/goque/internal/storages/dbtx"
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

	// Regression: a *sqlx.Tx attached upstream (via goque.WithTx) must
	// be stripped before BOTH the user's job factory AND the enqueue.
	// The periodic ticker outlives the caller; enrolling its writes in
	// the caller's tx would race the caller's Commit/Rollback. Mirrors
	// AsyncAddTaskToQueue behavior.
	//
	// Captures the ctx at two points (factory and AddTaskToQueue) and
	// asserts neither sees the tx. The ctx happens-before edge from
	// the test goroutine to the writes is established by Stop()
	// blocking on gracefulStoppedCh — race detector verifies.
	t.Run("strips tx from ctx", func(t *testing.T) {
		t.Parallel()

		var factoryCtx context.Context
		txAwareFactory := func(ctx context.Context) (*entity.Task, error) {
			factoryCtx = ctx
			return task, nil
		}

		job, err := NewJob(
			t.Name(),
			SchedulerFunc(func(t time.Time) time.Time {
				return t.Add(time.Hour)
			}),
			txAwareFactory,
			WithRunOnStart(),
		)
		require.NoError(t, err)

		manager := mock_periodicprocessor.NewMockTaskQueueManager(gomock.NewController(t))

		var enqueueCtx context.Context
		manager.EXPECT().
			AddTaskToQueue(gomock.Any(), task).
			DoAndReturn(func(ctx context.Context, _ *entity.Task) error {
				enqueueCtx = ctx
				return nil
			})

		tx := &sqlx.Tx{}
		ctxWithTx := dbtx.WithTx(ctx, tx)

		processor := NewProcessor(manager, job)
		err = processor.Run(ctxWithTx)
		require.NoError(t, err)
		<-time.After(500 * time.Millisecond)
		processor.Stop()

		require.NotNil(t, factoryCtx, "job factory must have been called")
		_, factoryHasTx := dbtx.TxFromContext(factoryCtx)
		require.False(t, factoryHasTx,
			"tx must be stripped before the user's job factory runs")

		require.NotNil(t, enqueueCtx, "AddTaskToQueue must have been called")
		_, enqueueHasTx := dbtx.TxFromContext(enqueueCtx)
		require.False(t, enqueueHasTx,
			"tx must be stripped before AddTaskToQueue")
	})
}
