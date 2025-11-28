package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestAdd(t *testing.T) {
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		payload := testPayload{Data: "test"}
		task := entity.NewTask("test", toJSON(t, payload))

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		equalTask(t, task, dbTask)
	})

	t.Run("failed payload", func(t *testing.T) {
		task := entity.NewTask("test", "invalid payload")

		err := storage.AddTask(ctx, task)
		require.EqualError(t, err, "pq: invalid input syntax for type json")
	})
}
