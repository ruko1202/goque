// Package xvalue provides configuration values that may be static or change at runtime.
package xvalue

// Value provides a configuration value that may change over time.
type Value[T any] interface {
	// Get returns the current value. It must be non-blocking, must not panic
	// and must be safe for concurrent use: processors call it on the hot path.
	Get() T
}

// StaticValue is a Value that never changes.
type StaticValue[T any] struct {
	value T
}

// NewStaticValue creates a Value that always returns v.
func NewStaticValue[T any](v T) *StaticValue[T] {
	return &StaticValue[T]{value: v}
}

// Get returns the static value.
func (v *StaticValue[T]) Get() T {
	return v.value
}
