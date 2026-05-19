// Package periodicprocessor provides cron-based periodic job scheduling.
package periodicprocessor

import (
	"context"
	"time"

	"github.com/ruko1202/goque/internal/entity"
)

// Scheduler calculates the next run time for a periodic job.
type Scheduler interface {
	Next(time.Time) time.Time
}

// TaskFactory creates a task for a scheduled run.
type TaskFactory func(ctx context.Context) (*entity.Task, error)

// JobOptions configures a periodic job.
type JobOptions func(*Job)

// Job describes a producer that periodically inserts regular queue tasks.
type Job struct {
	name       string
	schedule   Scheduler
	factory    TaskFactory
	runOnStart bool
}

// NewJob creates a periodic job from a schedule and task factory.
func NewJob(
	name string,
	schedule Scheduler,
	factory TaskFactory,
	opts ...JobOptions,
) (*Job, error) {
	job := &Job{
		name:     name,
		schedule: schedule,
		factory:  factory,
	}

	for _, opt := range opts {
		opt(job)
	}

	return job, nil
}

// NewCronJob creates a periodic job from a standard 5-field cron spec.
func NewCronJob(
	name string,
	cronSpec string,
	cronLocation *time.Location,
	factory TaskFactory,
	opts ...JobOptions,
) (*Job, error) {
	schedule, err := CronSchedule(cronSpec, cronLocation)
	if err != nil {
		return nil, err
	}

	return NewJob(name, schedule, factory, opts...)
}

// Name returns the periodic job name.
func (j *Job) Name() string {
	return j.name
}

func (j *Job) next(now time.Time) time.Time {
	return j.schedule.Next(now)
}

func (j *Job) create(ctx context.Context) (*entity.Task, error) {
	return j.factory(ctx)
}

func (j *Job) shouldRunOnStart() bool {
	return j.runOnStart
}
