package queueprocessor

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/pkg/generated/mocks/mock_storages"

	"github.com/ruko1202/goque/internal/entity"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type procMocks struct {
	taskStorage *mock_storages.MockTask
}

func initGoqueProcessorWithMocks(
	t *testing.T,
	taskType string,
	processor TaskProcessor,
	opts ...GoqueProcessorOpts,
) (*GoqueProcessor, *procMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)

	mocks := &procMocks{
		taskStorage: mock_storages.NewMockTask(ctrl),
	}
	goqueProc := NewGoqueProcessor(mocks.taskStorage, taskType, processor, opts...)

	return goqueProc, mocks
}

func defaultFetcherMock(mocks *procMocks, taskType string, tasks []*entity.Task) {
	gomock.InOrder(
		mocks.taskStorage.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, defaultFetchMaxTasks).
			Return(tasks, nil),
		mocks.taskStorage.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, defaultFetchMaxTasks).
			Return([]*entity.Task{}, nil).
			AnyTimes(),
	)
}
