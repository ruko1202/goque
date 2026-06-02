package entity

import (
	"fmt"

	"github.com/goccy/go-json"
)

// TypedTask represents a task with a payload decoded into the expected Go type.
type TypedTask[T any] struct {
	*Task
	Payload T
}

// NewTaskWithPayload creates a new task with a typed payload marshaled as JSON.
func NewTaskWithPayload[T any](taskType TaskType, payload T) (*Task, error) {
	return NewTaskWithPayloadAndExternalID[T](taskType, payload, "")
}

// NewTaskWithPayloadAndExternalID creates a new task with a typed payload and custom external ID.
func NewTaskWithPayloadAndExternalID[T any](taskType TaskType, payload T, externalID string) (*Task, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: encode %s task payload: %w", ErrPayloadMarshal, taskType, err)
	}

	return NewTaskWithExternalID(taskType, string(payloadJSON), externalID), nil
}
