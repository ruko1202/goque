package periodicprocessor

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// SchedulerFunc adapts a function to the Scheduler interface.
type SchedulerFunc func(time.Time) time.Time

// Next returns the next scheduled run time.
func (f SchedulerFunc) Next(t time.Time) time.Time {
	return f(t)
}

type cronSchedule struct {
	schedule Scheduler
	location *time.Location
}

// CronSchedule creates a schedule from a standard 5-field cron spec.
func CronSchedule(spec string, location *time.Location) (Scheduler, error) {
	if location == nil {
		return nil, errors.New("cron location is nil")
	}

	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return nil, fmt.Errorf("parse cron spec: %w", err)
	}

	return cronSchedule{
		schedule: schedule,
		location: location,
	}, nil
}

func (s cronSchedule) Next(t time.Time) time.Time {
	return s.schedule.Next(t.In(s.location))
}
