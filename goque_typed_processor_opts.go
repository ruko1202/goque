package goque

import "github.com/ruko1202/goque/internal/processors/queueprocessor"

// WithCancelTaskWhenPayloadDecodeError cancels typed tasks when payload decoding fails instead of retrying them.
func WithCancelTaskWhenPayloadDecodeError[T any]() TypedTaskProcessorOpt[T] {
	return queueprocessor.WithCancelTaskWhenPayloadDecodeError[T]()
}
