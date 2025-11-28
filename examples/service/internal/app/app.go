package app

import (
	"example/internal/config"

	"github.com/ruko1202/goque"
)

// Application represents the main application with all dependencies.
type Application struct {
	cfg          *config.Config
	queueManager goque.TaskQueueManager
}

// New creates a new Application instance with the provided Goque storage.
func New(cfg *config.Config, queueManager goque.TaskQueueManager) *Application {
	return &Application{
		cfg:          cfg,
		queueManager: queueManager,
	}
}
