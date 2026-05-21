package goque

import (
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

// TaskProcessor defines the interface for processing individual tasks.
type TaskProcessor = queueprocessor.TaskProcessor

// TaskProcessorFunc is a function type that implements the TaskProcessor interface.
type TaskProcessorFunc = queueprocessor.TaskProcessorFunc

// NoopTaskProcessor is a no-op task processor that does nothing and returns nil.
var NoopTaskProcessor = queueprocessor.NoopTaskProcessor

// ProcessorOpts is a function type for configuring GoqueProcessor options.
type ProcessorOpts = queueprocessor.GoqueProcessorOpts
