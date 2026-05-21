package queueprocessor

// GoqueTypedProcessorOpts configures a typed task processor adapter.
type GoqueTypedProcessorOpts[T any] func(*GoqueTypedProcessor[T])

// WithCancelTaskWhenPayloadDecodeError cancels typed tasks when payload decoding fails instead of retrying them.
func WithCancelTaskWhenPayloadDecodeError[T any]() GoqueTypedProcessorOpts[T] {
	return func(p *GoqueTypedProcessor[T]) {
		p.cancelTaskIfDecodePayloadError = true
	}
}
