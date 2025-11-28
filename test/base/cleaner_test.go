package base

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

func TestCleaner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		task := entity.NewTask(
			"test healer"+uuid.NewString(),
			toJSON(t, "test payload: "+uuid.NewString()),
		)
		pushToQueue(ctx, t, task)

		goq := goque.NewGoque(taskStorage)
		goq.RegisterProcessor(
			task.Type,
			queueprocessor.NoopTaskProcessor(),
			queueprocessor.WithTaskFetcherTick(10*time.Millisecond),
			queueprocessor.WithCleanerPeriod(10*time.Millisecond),
			queueprocessor.WithCleanerUpdatedAtTimeAgo(10*time.Millisecond),
			queueprocessor.WithCleanerTimeout(time.Second),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			t.Log("wait removing the task", task.ID)

			_, err := taskStorage.GetTask(ctx, task.ID)
			return errors.Is(err, sql.ErrNoRows)
		}, time.Second, time.Millisecond*50)
	})
}
