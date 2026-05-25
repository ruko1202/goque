package queuemanager

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mocks/mock_storages"
)

func TestTaskQueueManager_CancelTask(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		task       *entity.Task
		prepare    func(storage *mock_storages.MockTask, task *entity.Task)
		assertFunc func(t *testing.T, err error)
	}{
		"should_return_error_when_get_task_fails": {
			task: &entity.Task{ID: uuid.New(), Status: entity.TaskStatusNew},
			prepare: func(storage *mock_storages.MockTask, task *entity.Task) {
				storage.EXPECT().
					GetTask(gomock.Any(), task.ID).
					Return(nil, assert.AnError)
			},
			assertFunc: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		"should_not_update_task_when_status_is_terminal": {
			task: &entity.Task{ID: uuid.New(), Status: entity.TaskStatusDone},
			prepare: func(storage *mock_storages.MockTask, task *entity.Task) {
				storage.EXPECT().
					GetTask(gomock.Any(), task.ID).
					Return(task, nil)
			},
			assertFunc: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		"should_cancel_task_when_status_is_non_terminal": {
			task: &entity.Task{ID: uuid.New(), Status: entity.TaskStatusNew},
			prepare: func(storage *mock_storages.MockTask, task *entity.Task) {
				gomock.InOrder(
					storage.EXPECT().
						GetTask(gomock.Any(), task.ID).
						Return(task, nil),
					storage.EXPECT().
						UpdateTask(gomock.Any(), task.ID, task).
						DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
							assert.Equal(t, entity.TaskStatusCanceled, task.Status)
							return nil
						}),
				)
			},
			assertFunc: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
	}

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			storage := mock_storages.NewMockTask(ctrl)
			tt.prepare(storage, tt.task)

			manager := NewTaskQueueManager(storage)

			err := manager.CancelTask(context.Background(), tt.task.ID)
			tt.assertFunc(t, err)
		})
	}
}

// TestTaskQueueManager_WaitAsyncEnqueues_Drains verifies that
// WaitAsyncEnqueues blocks until every in-flight AsyncAddTaskToQueue
// goroutine has completed. Critical for graceful shutdown — without
// it Goque.Stop() returns while async writes are still in flight,
// leading to "sql: database is closed" errors when the caller closes
// the pool.
func TestTaskQueueManager_WaitAsyncEnqueues_Drains(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	storage := mock_storages.NewMockTask(ctrl)

	// AddTask blocks on this channel until the test releases it.
	// That way the goroutine is provably still in flight when
	// WaitAsyncEnqueues is called — the test fails (deadlock-style
	// timeout) if WaitAsyncEnqueues doesn't actually wait.
	release := make(chan struct{})
	completed := make(chan struct{})
	storage.EXPECT().
		AddTask(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *entity.Task) error {
			<-release
			close(completed)
			return nil
		})

	manager := NewTaskQueueManager(storage)
	manager.AsyncAddTaskToQueue(context.Background(), &entity.Task{
		ID:   uuid.New(),
		Type: "test",
	})

	// WaitAsyncEnqueues must not return before we release the storage
	// call.
	waitDone := make(chan struct{})
	go func() {
		manager.WaitAsyncEnqueues()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("WaitAsyncEnqueues returned before the in-flight async enqueue completed")
	case <-time.After(50 * time.Millisecond):
		// Expected: WaitAsyncEnqueues is correctly blocked.
	}

	close(release)
	<-completed

	select {
	case <-waitDone:
		// Expected: WaitAsyncEnqueues unblocks after the goroutine finishes.
	case <-time.After(time.Second):
		t.Fatal("WaitAsyncEnqueues did not unblock after the async enqueue completed")
	}
}
