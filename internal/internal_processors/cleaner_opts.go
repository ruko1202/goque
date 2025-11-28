// Package internalprocessors provides internal task processors for queue management and maintenance.
package internalprocessors

import (
	"time"

	"github.com/ruko1202/goque/internal/commonopts"
)

// GetCleanerOpts extracts cleaner-specific options from a list of internal processor options.
func GetCleanerOpts(opts []commonopts.InternalProcessorOpt) []QueueCleanerOpts {
	return commonopts.GetOpts[QueueCleanerOpts](opts, CleanerProcessorName)
}

// QueueCleanerOpts is a function type for configuring QueueCleaner options.
type QueueCleanerOpts func(*QueueCleaner)

// OptionType returns the option type identifier for cleaner options.
func (o QueueCleanerOpts) OptionType() string { return CleanerProcessorName }

// WithCleanerUpdatedAtTimeAgo sets the time threshold for considering tasks as old enough to be cleaned.
func WithCleanerUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) QueueCleanerOpts {
	return func(h *QueueCleaner) {
		h.updatedAtTimeAgo = updatedAtTimeAgo
	}
}
