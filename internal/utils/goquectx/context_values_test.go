package goquectx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithValue(t *testing.T) {
	t.Run("ok[default]", func(t *testing.T) {
		ctx := WithValue(context.Background(), "key", "value")
		ctx = WithValue(ctx, "keyint", 1)

		kv := Values(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "value", kv["key"])
		require.Equal(t, 1, kv["keyint"])
	})

	t.Run("ok[nested context]", func(t *testing.T) {
		ctx := WithValue(
			WithValue(context.Background(), "nested key", "nested value"),
			"root key", "root value",
		)

		kv := Values(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "nested value", kv["nested key"])
		require.Equal(t, "root value", kv["root key"])
	})

	t.Run("ok[multi]", func(t *testing.T) {
		ctx := WithValues(context.Background(), map[string]any{
			"key1": "value1",
			"key2": "value2",
		})

		kv := Values(ctx)
		require.Equal(t, 2, len(kv))
		require.Equal(t, "value1", kv["key1"])
		require.Equal(t, "value2", kv["key2"])
	})
}
