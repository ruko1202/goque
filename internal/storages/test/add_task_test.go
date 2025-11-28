package test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/storages"

	"github.com/ruko1202/goque/test/testutils"

	"github.com/ruko1202/goque/internal/storages/dbentity"
)

func TestAdd(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testAdd)
}

//nolint:thelper
func testAdd(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		payload := testutils.TestPayload{Data: "test"}
		task := entity.NewTask("test", testutils.ToJSON(t, payload))

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
	})

	t.Run("failed payload", func(t *testing.T) {
		t.Parallel()
		task := entity.NewTask("test", "invalid payload")

		err := storage.AddTask(ctx, task)
		require.ErrorIs(t, err, dbentity.ErrInvalidPayloadFormat)
	})

	t.Run("several externalID", func(t *testing.T) {
		t.Parallel()
		task := entity.NewTaskWithExternalID("test", testutils.ToJSON(t, "payload"), uuid.NewString())

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		err = storage.AddTask(ctx, task)
		require.ErrorIs(t, err, dbentity.ErrDuplicateTask)
	})
}
