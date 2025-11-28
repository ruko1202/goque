package internalprocessors

import "github.com/ruko1202/goque/internal/commonopts"

// Processor defines the interface for internal processors that can be tuned with options.
type Processor interface {
	Tune(opts []commonopts.InternalProcessorOpt)
}
