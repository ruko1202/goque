package entity

import (
	"errors"
)

var (
	// ErrDuplicateTask is returned when attempting to insert a task with a duplicate external ID.
	ErrDuplicateTask = errors.New("task already exists")
	// ErrInvalidPayloadFormat is returned when the task payload is not valid JSON.
	ErrInvalidPayloadFormat = errors.New("payload format is invalid. should be json")

	// ErrTaskCancel is returned when a task is canceled during processing.
	ErrTaskCancel = errors.New("task canceled")
	// ErrTaskTimeout is returned when task processing exceeds the timeout limit.
	ErrTaskTimeout = errors.New("task processing timeout")
)
