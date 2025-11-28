package test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/test/testutils"

	"github.com/ruko1202/goque/internal/storages"
)

func TestGet(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGet)
}

//nolint:thelper
func testGet(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		task, err := storage.GetTask(ctx, uuid.New())
		require.Nil(t, task)
		require.EqualError(t, err, sql.ErrNoRows.Error())
	})
}
