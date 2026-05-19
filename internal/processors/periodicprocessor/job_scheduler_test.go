package periodicprocessor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronSchedule(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		spec       string
		location   *time.Location
		assertFunc func(t *testing.T, schedule Scheduler, err error)
	}{
		"should_return_error_when_location_is_nil": {
			spec:     "* * * * *",
			location: nil,
			assertFunc: func(t *testing.T, schedule Scheduler, err error) {
				t.Helper()

				require.Error(t, err)
				assert.Nil(t, schedule)
			},
		},
		"should_return_error_when_cron_spec_is_invalid": {
			spec:     "invalid",
			location: time.UTC,
			assertFunc: func(t *testing.T, schedule Scheduler, err error) {
				t.Helper()

				require.Error(t, err)
				assert.Nil(t, schedule)
			},
		},
		"should_return_schedule_when_cron_spec_is_valid": {
			spec:     "* * * * *",
			location: time.UTC,
			assertFunc: func(t *testing.T, schedule Scheduler, err error) {
				t.Helper()

				require.NoError(t, err)
				require.NotNil(t, schedule)

				now := time.Now().UTC()
				nextRun := schedule.Next(now)

				assert.True(t, nextRun.After(now))
				assert.LessOrEqual(t, nextRun.Sub(now), time.Minute)
				assert.Equal(t, 0, nextRun.Second())
			},
		},
	}

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			schedule, err := CronSchedule(tt.spec, tt.location)
			tt.assertFunc(t, schedule, err)
		})
	}
}
