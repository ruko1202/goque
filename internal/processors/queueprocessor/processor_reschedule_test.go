package queueprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

func TestGoqueProcessor_Reschedule(t *testing.T) {
	t.Parallel()

	for _, after := range []time.Duration{time.Minute, -time.Minute} {
		t.Run(after.String(), func(t *testing.T) {
			t.Parallel()
			taskType := "type[reschedule " + after.String() + "]"

			task := newTestTask(taskType)
			task.Attempts = 2
			task.Errors = lo.ToPtr("attempt 2: boom\n")

			updated := make(chan entity.Task, 1)
			goqueProc, mocks := initGoqueProcessorWithMocks(t, taskType,
				TaskProcessorFunc(func(context.Context, *entity.Task) error {
					return fmt.Errorf("status is not final: %w", entity.RescheduleTask(after))
				}),
				WithTaskFetcherTick(xvalue.NewStaticValue(10*time.Millisecond)),
			)
			defaultFetcherMock(mocks, taskType, []*entity.Task{task})
			mocks.taskStorage.EXPECT().UpdateTask(gomock.Any(), task.ID, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, task *entity.Task) error {
					if task.Status != entity.TaskStatusProcessing {
						updated <- *task
					}
					return nil
				}).
				Times(2)

			before := xtime.Now()
			require.NoError(t, goqueProc.Run(testCtx(t)))
			defer goqueProc.Stop()
			got := recv(t, updated)

			assert.Equal(t, entity.TaskStatusNew, got.Status)
			assert.Equal(t, int32(2), got.Attempts)
			assert.Equal(t, "attempt 2: boom\n", lo.FromPtr(got.Errors))
			assert.WithinDuration(t, before.Add(max(after, 0)), got.NextAttemptAt, time.Second)
		})
	}
}
