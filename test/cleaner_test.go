package test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal"
	"github.com/ruko1202/goque/internal/entity"
	internalprocessors "github.com/ruko1202/goque/internal/internal_processors"
	"github.com/ruko1202/goque/internal/processor"
)

func TestCleaner(t *testing.T) {
	ctx := context.Background()

	task := entity.NewTask(
		"test healer"+uuid.NewString(),
		toJSON(t, "test payload: "+uuid.NewString()),
	)
	pushToQueue(ctx, t, task)

	goque := internal.NewGoque(taskStorage)
	goque.TuneCleanerProcessor(
		processor.WithTaskFetcherTick(100*time.Millisecond),
		internalprocessors.WithCleanerUpdatedAtTimeAgo(100*time.Millisecond),
	)
	goque.RegisterProcessor(
		task.Type,
		processor.NoopTaskProcessor(),
		processor.WithTaskFetcherTick(100*time.Millisecond),
	)
	goque.Run(ctx)
	defer goque.Stop()

	require.Eventually(t, func() bool {
		t.Log("wait removing the task", task.ID)

		_, err := taskStorage.GetTask(ctx, task.ID)
		return errors.Is(err, sql.ErrNoRows)
	}, time.Second*10, time.Millisecond*500)
}
