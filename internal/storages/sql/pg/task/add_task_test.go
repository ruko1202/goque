package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		payload := testPayload{Data: "test"}
		task := entity.NewTask("test", toJSON(t, payload))

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		equalTask(t, task, dbTask)
	})

	t.Run("failed payload", func(t *testing.T) {
		t.Parallel()
		task := entity.NewTask("test", "invalid payload")

		err := storage.AddTask(ctx, task)
		require.ErrorIs(t, err, ErrInvalidPayloadFormat)
	})

	t.Run("several externalID", func(t *testing.T) {
		t.Parallel()
		task := entity.NewTaskWithExternalID("test", toJSON(t, "payload"), uuid.NewString())

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		err = storage.AddTask(ctx, task)
		require.ErrorIs(t, err, ErrDuplicateTask)
	})
}
