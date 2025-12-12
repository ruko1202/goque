package entity

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// Metadata represents arbitrary key-value data associated with a task for tracking and context.
type Metadata map[string]any

// NewMetadataFromJSON deserializes a JSON string into a Metadata map.
func NewMetadataFromJSON(ctx context.Context, metadata *string) Metadata {
	metadataMap := make(map[string]any)

	err := json.Unmarshal([]byte(lo.FromPtr(metadata)), &metadataMap)
	if err != nil {
		xlog.Error(ctx, "unmarshal metadata", zap.Error(err))
	}

	return metadataMap
}

// Merge combines the current metadata with another metadata map, with values from the parameter taking precedence.
func (m Metadata) Merge(metadata Metadata) Metadata {
	return lo.Assign(m, metadata)
}

// ToJSON serializes the metadata map into a JSON string.
func (m Metadata) ToJSON(ctx context.Context) string {
	metadata, err := json.Marshal(m)
	if err != nil {
		xlog.Error(ctx, "marshaling metadata", zap.Error(err))
		return ""
	}

	return string(metadata)
}
