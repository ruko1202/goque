// Package app contains the main application logic.
package app

import (
	"example/internal/config"

	"github.com/jmoiron/sqlx"

	"github.com/ruko1202/goque"
)

// Application represents the main application with all dependencies.
type Application struct {
	cfg          *config.Config
	queueManager goque.TaskQueueManager
	// db is the shared *sqlx.DB used both by goque (already wrapped
	// inside queueManager) and by this service's domain writes
	// (e.g. the orders table used by the transactional-outbox
	// example endpoint). Sharing the same pool is what makes the
	// outbox pattern work: a single tx opened on `db` is honored
	// by goque via WithTx and by the domain INSERT side-by-side.
	db *sqlx.DB
}

// New creates a new Application instance with the provided Goque storage.
func New(cfg *config.Config, queueManager goque.TaskQueueManager, db *sqlx.DB) *Application {
	return &Application{
		cfg:          cfg,
		queueManager: queueManager,
		db:           db,
	}
}
