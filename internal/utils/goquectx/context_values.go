package goquectx

import (
	"context"

	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
)

type goqueContex string

const goqueContextKeys goqueContex = "qoguectxkeys"

func ContextWithValue(ctx context.Context, key string, value any) context.Context {
	goqueKeys := goqueKeysFromContext(ctx)
	goqueKeys = append(goqueKeys, key)

	ctx = context.WithValue(ctx, key, value)
	return context.WithValue(ctx, goqueContextKeys, goqueKeys)
}

func ContextWithValues(ctx context.Context, kv entity.Metadata) context.Context {
	for key, value := range kv {
		ctx = ContextWithValue(ctx, key, value)
	}
	return ctx
}

func ValuesFromContext(ctx context.Context) entity.Metadata {
	goqueKeys := goqueKeysFromContext(ctx)

	return lo.SliceToMap(goqueKeys, func(key string) (string, any) {
		return key, ctx.Value(key)
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
