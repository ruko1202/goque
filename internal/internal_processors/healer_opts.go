// Package internalprocessors provides internal task processors for queue management and maintenance.
package internalprocessors

import (
	"time"

	"github.com/ruko1202/goque/internal/commonopts"
)

// GetHealerOpts extracts healer-specific options from a list of internal processor options.
func GetHealerOpts(opts []commonopts.InternalProcessorOpt) []QueueHealerOpts {
	return commonopts.GetOpts[QueueHealerOpts](opts, Healer)
}

// QueueHealerOpts is a function type for configuring QueueHealer options.
type QueueHealerOpts func(*QueueHealer)

// OptionType returns the option type identifier for healer options.
func (o QueueHealerOpts) OptionType() string { return Healer }

// WithHealerUpdatedAtTimeAgo sets the time threshold for considering a task as stuck.
func WithHealerUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) QueueHealerOpts {
	return func(h *QueueHealer) {
		h.updatedAtTimeAgo = updatedAtTimeAgo
	}
}

// WithHealerMaxTasks sets the maximum number of tasks the healer will process in one batch.
func WithHealerMaxTasks(maxTasks int64) QueueHealerOpts {
	return func(h *QueueHealer) {
		h.maxTasks = maxTasks
	}
}
