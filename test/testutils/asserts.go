// Package testutils provides testing utilities for the goque project.
package testutils

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

const (
	timeDelta = 5 * time.Minute
)

// EqualTask asserts that two tasks are equal, comparing all fields with appropriate tolerances.
func EqualTask(t *testing.T, expected, actual *entity.Task) {
	t.Helper()

	require.Equal(t, expected.ID.String(), actual.ID.String())
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.ExternalID, actual.ExternalID)
	require.Equal(t, FromJSON(t, expected.Payload), FromJSON(t, actual.Payload))
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Attempts, actual.Attempts)
	require.Equal(t, lo.FromPtr(expected.Errors), lo.FromPtr(actual.Errors))
	AssertTimeInWithDelta(t, expected.CreatedAt, actual.CreatedAt, timeDelta)
	if expected.UpdatedAt == nil {
		require.Nil(t, actual.UpdatedAt)
	} else {
		AssertTimeInWithDelta(t, lo.FromPtr(expected.UpdatedAt), lo.FromPtr(actual.UpdatedAt), timeDelta)
	}
	AssertTimeInWithDelta(t, expected.NextAttemptAt, actual.NextAttemptAt, timeDelta)
}

// AssertTimeInWithDelta asserts that two time values are equal within the specified delta tolerance.
func AssertTimeInWithDelta(t *testing.T, expected, actual time.Time, expectedDelta time.Duration) {
	t.Helper()
	actualDelta := expected.Sub(actual).Abs()
	isEqual := expected.Equal(actual) || actualDelta <= expectedDelta.Abs()
	require.True(t, isEqual,
		"expected time: %s\nactual time:   %s\n"+
			"expected delta: %s\nactual delta: %s\n",
		expected, actual,
		expectedDelta.Abs(), actualDelta,
	)
}
