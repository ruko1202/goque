package xvalue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticValue(t *testing.T) {
	t.Parallel()

	v := NewStaticValue(42)
	assert.Equal(t, 42, v.Get())

	var _ Value[int] = v
}
