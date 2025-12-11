// Package goquectx provides utilities for managing task metadata within context values.
package goquectx

import (
	"context"

	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
)

type goqueCtcKey string

const goqueContextKeys goqueCtcKey = "qoguectxkeys"

// WithValue adds a single key-value pair to the context for task metadata tracking.
func WithValue(ctx context.Context, key string, value any) context.Context {
	return contextWithValues(ctx, entity.Metadata{key: value})
}

// WithValues adds multiple key-value pairs to the context for task metadata tracking.
func WithValues(ctx context.Context, kv entity.Metadata) context.Context {
	return contextWithValues(ctx, kv)
}

func contextWithValues(ctx context.Context, kv entity.Metadata) context.Context {
	goqueKeys := goqueKeysFromContext(ctx)
	goqueKeys = append(goqueKeys, lo.Keys(kv)...)

	for key, value := range kv {
		ctx = context.WithValue(ctx, goqueCtcKey(key), value)
	}

	return context.WithValue(ctx, goqueContextKeys, goqueKeys)
}

// ValueByKey retrieves stored value from the context by key.
func ValueByKey(ctx context.Context, key string) any {
	return ctx.Value(goqueCtcKey(key))
}

// Values retrieves all stored metadata values from the context.
func Values(ctx context.Context) entity.Metadata {
	goqueKeys := goqueKeysFromContext(ctx)

	return lo.SliceToMap(goqueKeys, func(key string) (string, any) {
		return key, ctx.Value(goqueCtcKey(key))
	})
}

func goqueKeysFromContext(ctx context.Context) []string {
	val := ctx.Value(goqueContextKeys)
	if val == nil {
		return []string{}
	}

	goqueKeys, ok := val.([]string)
	if !ok {
		return []string{}
	}

	return goqueKeys
}
