package goquectx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithValue(t *testing.T) {
	t.Run("ok[default]", func(t *testing.T) {
		ctx := ContextWithValue(context.Background(), "key", "value")
		ctx = ContextWithValue(ctx, "keyint", 1)

		kv := ValuesFromContext(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "value", kv["key"])
		require.Equal(t, 1, kv["keyint"])
	})

	t.Run("ok[nested context]", func(t *testing.T) {
		ctx := ContextWithValue(
			ContextWithValue(context.Background(), "nested key", "nested value"),
			"root key", "root value",
		)

		kv := ValuesFromContext(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "nested value", kv["nested key"])
		require.Equal(t, "root value", kv["root key"])
	})

	t.Run("ok[multi]", func(t *testing.T) {
		ctx := ContextWithValues(context.Background(), map[string]any{
			"key1": "value1",
			"key2": "value2",
		})

		kv := ValuesFromContext(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "value1", kv["key1"])
		require.Equal(t, "value2", kv["key2"])
	})
}
