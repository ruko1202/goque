package entity

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripPayload struct {
	OrderID string `json:"order_id"`
	Amount  int    `json:"amount"`
}

func TestNewTaskWithPayload(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		p := roundTripPayload{OrderID: "o-1", Amount: 42}

		task, err := NewTaskWithPayload("order_confirmation", p)
		require.NoError(t, err)
		require.NotNil(t, task)
		require.Equal(t, TaskType("order_confirmation"), task.Type)
		require.Equal(t, TaskStatusNew, task.Status)

		// Payload survives the JSON round-trip.
		var got roundTripPayload
		require.NoError(t, json.Unmarshal([]byte(task.Payload), &got))
		require.Equal(t, p, got)
	})

	t.Run("marshal failure wraps ErrPayloadMarshal", func(t *testing.T) {
		t.Parallel()

		// chan can't be JSON-marshaled — exercises the error branch.
		task, err := NewTaskWithPayload("t", map[string]any{"ch": make(chan int)})
		require.Nil(t, task)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrPayloadMarshal),
			"error must wrap ErrPayloadMarshal so callers can detect via errors.Is, got: %v", err)
	})
}

func TestNewTaskWithPayloadAndExternalID(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		p := roundTripPayload{OrderID: "o-7"}

		task, err := NewTaskWithPayloadAndExternalID("t", p, "external-key-7")
		require.NoError(t, err)
		require.Equal(t, "external-key-7", task.ExternalID)
	})

	t.Run("marshal failure surfaces wrapped error", func(t *testing.T) {
		t.Parallel()
		task, err := NewTaskWithPayloadAndExternalID("t", map[string]any{"ch": make(chan int)}, "x")
		require.Nil(t, task)
		require.True(t, errors.Is(err, ErrPayloadMarshal))
	})
}
