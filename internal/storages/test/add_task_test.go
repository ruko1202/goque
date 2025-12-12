package test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/test/testutils"
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
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		payload := testutils.TestPayload{Data: "test"}
		task := entity.NewTask("test", testutils.ToJSON(t, payload))

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
		require.Equal(t, goque.Metadata{"testname": t.Name()}, dbTask.Metadata)
	})

	t.Run("failed payload", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		task := entity.NewTask("test", "invalid payload")

		err := storage.AddTask(ctx, task)
		require.ErrorIs(t, err, entity.ErrInvalidPayloadFormat)
	})

	t.Run("several externalID", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		task := entity.NewTaskWithExternalID("test", testutils.ToJSON(t, "payload"), uuid.NewString())

		err := storage.AddTask(ctx, task)
		require.NoError(t, err)

		err = storage.AddTask(ctx, task)
		require.ErrorIs(t, err, entity.ErrDuplicateTask)
	})
}
