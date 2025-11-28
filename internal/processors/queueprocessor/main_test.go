package queueprocessor

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"

	mock_queueprocessor "github.com/ruko1202/goque/internal/pkg/generated/mocks/mock_processors/queueprocessor"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type procMocks struct {
	taskService *mock_queueprocessor.MockTaskStorage
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
		taskService: mock_queueprocessor.NewMockTaskStorage(ctrl),
	}
	goqueProc := NewGoqueProcessor(mocks.taskService, taskType, processor, opts...)

	return goqueProc, mocks
}

func defaultFetcherMock(mocks *procMocks, taskType string, tasks []*entity.Task) {
	gomock.InOrder(
		mocks.taskService.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, defaultFetchMaxTasks).
			Return(tasks, nil),
		mocks.taskService.EXPECT().
			GetTasksForProcessing(gomock.Any(), taskType, defaultFetchMaxTasks).
			Return([]*entity.Task{}, nil).
			AnyTimes(),
	)
}
