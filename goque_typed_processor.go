package goque

import "github.com/ruko1202/goque/internal/processors/queueprocessor"

// TypedTaskProcessor defines the interface for processing typed task payloads.
type TypedTaskProcessor[T any] = queueprocessor.TypedTaskProcessor[T]

// TypedTaskProcessorFunc is a function type that implements the TypedTaskProcessor interface.
type TypedTaskProcessorFunc[T any] = queueprocessor.TypedTaskProcessorFunc[T]

// TypedTaskProcessorOpt configures a typed task processor adapter.
type TypedTaskProcessorOpt[T any] = queueprocessor.GoqueTypedProcessorOpts[T]

// NewTypedTaskProcessor wraps a typed task processor for use with RegisterProcessor.
func NewTypedTaskProcessor[T any](processor TypedTaskProcessor[T], opts ...TypedTaskProcessorOpt[T]) TaskProcessor {
	return queueprocessor.NewTypedTaskProcessor[T](processor, opts...)
}
