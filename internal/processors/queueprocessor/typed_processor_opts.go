package queueprocessor

type GoqueTypedProcessorOpts[T any] func(*GoqueTypedProcessor[T])

func WithCancelTaskWhenPayloadDecodeError[T any]() GoqueTypedProcessorOpts[T] {
	return func(p *GoqueTypedProcessor[T]) {
		p.cancelTaskIfDecodePayloadError = true
	}
}
