package entity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadata_ToJSON(t *testing.T) {
	meta := Metadata{
		"key": "value",
	}

	j := meta.ToJSON(context.Background())
	require.Equal(t, `{"key":"value"}`, j)
}
