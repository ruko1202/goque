// Package commonopts provides common option types for configuring processors.
package commonopts

import "github.com/samber/lo"

// InternalProcessorOpt is an interface for processor configuration options.
type InternalProcessorOpt interface {
	OptionType() string
}

// GetOpts extracts options of a specific type from a list of internal processor options.
func GetOpts[T any](opts []InternalProcessorOpt, optType string) []T {
	filteredOpts := lo.Filter(opts, func(item InternalProcessorOpt, _ int) bool {
		return item.OptionType() == optType
	})

	return lo.FilterMap(filteredOpts, func(item InternalProcessorOpt, _ int) (T, bool) {
		opt, ok := item.(T)
		return opt, ok
	})
}
