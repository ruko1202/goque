package entity

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestMetadata_ToJSON(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		meta := Metadata{"key": "value"}
		j := meta.ToJSON(context.Background())
		require.Equal(t, `{"key":"value"}`, j)
	})

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()
		meta := Metadata{}
		j := meta.ToJSON(context.Background())
		require.Equal(t, `{}`, j)
	})

	t.Run("nil map serializes as null", func(t *testing.T) {
		t.Parallel()
		var meta Metadata
		j := meta.ToJSON(context.Background())
		require.Equal(t, `null`, j)
	})

	t.Run("non-serialisable value returns empty", func(t *testing.T) {
		t.Parallel()
		// chan can't be JSON-marshaled — ToJSON logs and returns "".
		meta := Metadata{"ch": make(chan int)}
		j := meta.ToJSON(context.Background())
		require.Equal(t, "", j)
	})
}

func TestNewMetadataFromJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		t.Parallel()
		raw := `{"k":"v","n":1}`
		meta := NewMetadataFromJSON(context.Background(), &raw)
		require.Equal(t, "v", meta["k"])
		require.EqualValues(t, 1, meta["n"])
	})

	t.Run("nil pointer yields empty metadata", func(t *testing.T) {
		t.Parallel()
		meta := NewMetadataFromJSON(context.Background(), nil)
		require.NotNil(t, meta)
		require.Empty(t, meta)
	})

	t.Run("empty string yields empty metadata", func(t *testing.T) {
		t.Parallel()
		s := ""
		meta := NewMetadataFromJSON(context.Background(), &s)
		require.NotNil(t, meta)
		require.Empty(t, meta)
	})

	t.Run("malformed json logs and returns empty", func(t *testing.T) {
		t.Parallel()
		s := "not-json"
		meta := NewMetadataFromJSON(context.Background(), &s)
		// Implementation returns an initialized (empty) map; we don't
		// assert on the log, just that the call doesn't panic and the
		// caller gets a usable map back.
		require.NotNil(t, meta)
	})
}

func TestMetadata_Merge(t *testing.T) {
	t.Parallel()

	t.Run("incoming overrides existing", func(t *testing.T) {
		t.Parallel()
		base := Metadata{"a": 1, "b": 2}
		extra := Metadata{"b": 99, "c": 3}

		merged := base.Merge(extra)

		require.EqualValues(t, 1, merged["a"])
		require.EqualValues(t, 99, merged["b"], "extra must override base")
		require.EqualValues(t, 3, merged["c"])
	})

	t.Run("empty parameter returns base copy", func(t *testing.T) {
		t.Parallel()
		base := Metadata{"a": 1}
		merged := base.Merge(Metadata{})
		require.EqualValues(t, 1, merged["a"])
	})

	t.Run("nil parameter does not panic", func(t *testing.T) {
		t.Parallel()
		base := Metadata{"a": 1}
		require.NotPanics(t, func() {
			merged := base.Merge(nil)
			require.EqualValues(t, 1, merged["a"])
		})
	})
}

func TestMetadata_RoundTrip(t *testing.T) {
	t.Parallel()

	// ToJSON → NewMetadataFromJSON must preserve string values.
	// Numeric values come back as float64 (Go's default for json.Unmarshal
	// into map[string]any), so we only round-trip strings here.
	original := Metadata{"trace_id": "abc-123", "user": "alice"}
	j := original.ToJSON(context.Background())
	require.NotEmpty(t, j)

	parsed := NewMetadataFromJSON(context.Background(), lo.ToPtr(j))
	require.Equal(t, "abc-123", parsed["trace_id"])
	require.Equal(t, "alice", parsed["user"])
}
