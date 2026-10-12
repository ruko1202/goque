package goque_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ruko1202/goque"
)

func TestValueReExports(t *testing.T) {
	t.Parallel()

	var static goque.Value[int64] = goque.NewStaticValue[int64](100)
	assert.EqualValues(t, 100, static.Get())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	dynamic := goque.NewDynamicValue(ctx, 1, func(context.Context) (int, error) { return 20, nil }, time.Minute)
	assert.Equal(t, 20, dynamic.Get())

	// The re-exported values plug straight into processor options.
	_ = goque.WithWorkersCount(dynamic)
	_ = goque.WithTaskFetcherMaxTasks(static)
}
