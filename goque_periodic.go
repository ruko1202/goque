package goque

import (
	"github.com/ruko1202/goque/internal/processors/periodicprocessor"
)

type (
	// PeriodicSchedule calculates the next run time for a periodic job.
	PeriodicSchedule = periodicprocessor.Scheduler
	// PeriodicSchedulerFunc wrap PeriodicSchedule.
	PeriodicSchedulerFunc = periodicprocessor.SchedulerFunc
	// PeriodicJobFactory creates a task for a scheduled run.
	PeriodicJobFactory = periodicprocessor.TaskFactory
	// PeriodicJobOpts configures a periodic job.
	PeriodicJobOpts = periodicprocessor.JobOptions
	// PeriodicJob describes a producer that periodically inserts regular queue tasks.
	PeriodicJob = periodicprocessor.Job
)

var (
	// NewPeriodicJob creates a periodic job from a schedule and task factory.
	NewPeriodicJob = periodicprocessor.NewJob
	// NewCronJob creates a periodic job from a standard 5-field cron spec.
	NewCronJob = periodicprocessor.NewCronJob
	// CronSchedule creates a schedule from a standard 5-field cron spec.
	CronSchedule = periodicprocessor.CronSchedule
	// WithPeriodicJobRunOnStart makes a periodic job enqueue one task when the scheduler starts.
	WithPeriodicJobRunOnStart = periodicprocessor.WithRunOnStart
)
