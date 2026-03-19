package entity

import (
	"bytes"
	"context"

	"github.com/goccy/go-json"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/utils/xpool"
)

// Metadata represents arbitrary key-value data associated with a task for tracking and context.
type Metadata map[string]any

// NewMetadataFromJSON deserializes a JSON string into a Metadata map.
func NewMetadataFromJSON(ctx context.Context, metadata *string) Metadata {
	metadataMap := make(map[string]any)

	err := json.Unmarshal([]byte(lo.FromPtr(metadata)), &metadataMap)
	if err != nil {
		xlog.Error(ctx, "unmarshal metadata", xfield.Error(err))
	}

	return metadataMap
}

// Merge combines the current metadata with another metadata map, with values from the parameter taking precedence.
func (m Metadata) Merge(metadata Metadata) Metadata {
	return lo.Assign(m, metadata)
}

// ToJSON serializes the metadata map into a JSON string.
func (m Metadata) ToJSON(ctx context.Context) string {
	buf := xpool.AcquireBuffer()
	defer xpool.ReleaseBuffer(buf)

	err := json.NewEncoder(buf).Encode(m)
	if err != nil {
		xlog.Error(ctx, "marshaling metadata", xfield.Error(err))
		return ""
	}

	return string(bytes.TrimSpace(buf.Bytes()))
}
