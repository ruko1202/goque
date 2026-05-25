package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	t.Parallel()

	task := NewTask("email", `{"to":"x@y"}`)

	require.NotEqual(t, uuid.Nil, task.ID)
	require.Equal(t, TaskType("email"), task.Type)
	require.Equal(t, `{"to":"x@y"}`, task.Payload)
	require.Equal(t, TaskStatusNew, task.Status)
	require.True(t, len(task.ExternalID) > len("internal-"), "ExternalID must include UUID suffix")
	require.Contains(t, task.ExternalID, "internal-")
	require.False(t, task.CreatedAt.IsZero())
	require.Equal(t, task.CreatedAt, task.NextAttemptAt, "NextAttemptAt should default to CreatedAt")
	require.Nil(t, task.UpdatedAt)
	require.Equal(t, int32(0), task.Attempts)
	require.Nil(t, task.Errors)
}

func TestNewTaskWithExternalID(t *testing.T) {
	t.Parallel()

	task := NewTaskWithExternalID("email", `{}`, "order-42")

	require.Equal(t, "order-42", task.ExternalID)
	require.Equal(t, TaskType("email"), task.Type)
	require.Equal(t, TaskStatusNew, task.Status)
}

func TestTask_AddError(t *testing.T) {
	t.Parallel()

	t.Run("nil error is a no-op", func(t *testing.T) {
		t.Parallel()
		task := NewTask("t", "{}")
		task.AddError(nil)
		require.Nil(t, task.Errors)
	})

	t.Run("first error produces single newline-terminated entry", func(t *testing.T) {
		t.Parallel()
		task := NewTask("t", "{}")
		task.Attempts = 1
		task.AddError(errors.New("boom"))

		require.NotNil(t, task.Errors)
		// Format is consumed by humans reading task.errors in the DB.
		// Pin the exact shape: "attempt N: msg\n" — anything else is a
		// breaking change for ops dashboards.
		require.Equal(t, "attempt 1: boom\n", *task.Errors)
	})

	t.Run("subsequent errors append with newline separator", func(t *testing.T) {
		t.Parallel()
		task := NewTask("t", "{}")
		task.Attempts = 1
		task.AddError(errors.New("first"))
		task.Attempts = 2
		task.AddError(errors.New("second"))

		require.NotNil(t, task.Errors)
		// Two distinct lines, each ending in \n. If a future refactor
		// drops the trailing \n the lines collapse into one and log
		// parsers break.
		require.Equal(t, "attempt 1: first\nattempt 2: second\n", *task.Errors)
	})
}

func TestTask_IsInTerminalState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   TaskStatus
		terminal bool
	}{
		{TaskStatusNew, false},
		{TaskStatusPending, false},
		{TaskStatusProcessing, false},
		{TaskStatusError, false},
		{TaskStatusDone, true},
		{TaskStatusCanceled, true},
		{TaskStatusAttemptsLeft, true},
		{"unknown_status", false},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			task := &Task{Status: tc.status}
			require.Equal(t, tc.terminal, task.IsInTerminalState())
		})
	}
}
