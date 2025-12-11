package entity

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type Metadata map[string]any

func NewMetadataFromJSON(ctx context.Context, metadata *string) Metadata {
	metadataMap := make(map[string]any)

	err := json.Unmarshal([]byte(lo.FromPtr(metadata)), &metadataMap)
	if err != nil {
		xlog.Error(ctx, "unmarshal metadata", zap.Error(err))
	}

	return metadataMap
}

func (m Metadata) Merge(metadata Metadata) Metadata {
	return lo.Assign(m, metadata)
}

func (m Metadata) ToJSON(ctx context.Context) string {
	metadata, err := json.Marshal(m)
	if err != nil {
		xlog.Error(ctx, "marshaling metadata", zap.Error(err))
		return ""
	}

	return string(metadata)
}
