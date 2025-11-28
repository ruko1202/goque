package task

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		task, err := storage.GetTask(ctx, uuid.New())
		require.Nil(t, task)
		require.EqualError(t, err, sql.ErrNoRows.Error())
	})
}
