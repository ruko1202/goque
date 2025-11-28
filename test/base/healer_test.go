package base

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/utils/xtime"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/processors/internalprocessors"
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
	taskstorage "github.com/ruko1202/goque/internal/storages/sql/pg/task"
)

func TestHealer(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		taskType := "test healer" + uuid.NewString()

		expectedCurredTasks := make([]*entity.Task, 0)
		for _, status := range []entity.TaskStatus{
			entity.TaskStatusPending,
			entity.TaskStatusProcessing,
		} {
			task := entity.NewTask(
				taskType,
				toJSON(t, "test payload: "+uuid.NewString()),
			)
			task.Status = status
			task.AddError(errors.New("processing error"))
			task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-2 * time.Hour))
			expectedCurredTasks = append(expectedCurredTasks, task)

			pushToQueue(ctx, t, task)
		}

		goq := goque.NewGoque(taskStorage)
		goq.RegisterProcessor(
			taskType,
			queueprocessor.NoopTaskProcessor(),
			queueprocessor.WithHealerPeriod(10*time.Millisecond),
			queueprocessor.WithHealerUpdatedAtTimeAgo(time.Hour),
			queueprocessor.WithHealerTimeout(time.Second),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			tasks, err := taskStorage.GetTasks(ctx, &taskstorage.GetTasksFilter{
				TaskType: lo.ToPtr(taskType),
			}, 100)
			require.NoError(t, err)
			require.Equal(t, len(expectedCurredTasks), len(tasks))

			tasksM := lo.SliceToMap(tasks, func(item *entity.Task) (uuid.UUID, *entity.Task) {
				return item.ID, item
			})

			curredTasks := 0
			for _, expectedTasks := range expectedCurredTasks {
				actualTask, ok := tasksM[expectedTasks.ID]
				require.True(t, ok)
				t.Log(actualTask.ID, "wait task status:", entity.TaskStatusError, "| actual status:", actualTask.Status)
				t.Log(actualTask.ID, "wait task errors:", internalprocessors.ErrTaskIsFrozen, "| actual errors:", lo.FromPtr(actualTask.Errors))

				if actualTask.Status == entity.TaskStatusError &&
					strings.Contains(lo.FromPtr(actualTask.Errors), internalprocessors.ErrTaskIsFrozen.Error()) {
					curredTasks++
				}
			}

			return curredTasks == len(expectedCurredTasks)
		}, time.Second, time.Millisecond*50)
	})
}
