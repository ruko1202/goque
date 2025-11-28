package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		task, err := storage.GetTask(ctx, uuid.New())
		require.Nil(t, task)
		require.EqualError(t, err, "sql: no rows in result set")
	})
}
