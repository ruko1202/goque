package entity

import (
	"errors"
	"time"
)

var (
	// ErrDuplicateTask is returned when attempting to insert a task with a duplicate external ID.
	ErrDuplicateTask = errors.New("task already exists")
	// ErrInvalidPayloadFormat is returned when the task payload is not valid JSON.
	ErrInvalidPayloadFormat = errors.New("payload format is invalid. should be json")
	// ErrPayloadUnmarshal is returned when a typed task payload cannot be unmarshaled from JSON.
	ErrPayloadUnmarshal = errors.New("payload unmarshal")
	// ErrPayloadMarshal is returned when a typed task payload cannot be marshaled to JSON.
	ErrPayloadMarshal = errors.New("payload marshal")

	// ErrTaskCancel is returned when a task is canceled during processing.
	ErrTaskCancel = errors.New("task canceled")
	// ErrTaskTimeout is returned when task processing exceeds the timeout limit.
	ErrTaskTimeout = errors.New("task processing timeout")
	// ErrTaskReschedule is matched by every error created with RescheduleTask.
	ErrTaskReschedule = errors.New("task rescheduled")
	// ErrTaskVersionConflict is returned when a task write is rejected because the
	// stored version no longer matches the writer's Task.Version: another writer
	// (healer, cancel, reset or another worker) changed the task, or the task was deleted.
	ErrTaskVersionConflict = errors.New("task version conflict")
	// ErrTaskAttemptsExhausted is passed to after-processing hooks when a fetched task
	// already used all its attempts (for example, it was cured by the healer repeatedly)
	// and is moved to attempts_left without being processed.
	ErrTaskAttemptsExhausted = errors.New("attempts exhausted")
	// ErrTaskPanic fails a task whose processing panicked.
	ErrTaskPanic = errors.New("task processor panicked")
)

// RescheduleError carries the delay requested by RescheduleTask.
type RescheduleError struct {
	After time.Duration
}

// RescheduleTask returns an error that puts the task back to the queue after the given delay.
func RescheduleTask(after time.Duration) error {
	return &RescheduleError{After: after}
}

func (e *RescheduleError) Error() string { return ErrTaskReschedule.Error() }
func (e *RescheduleError) Unwrap() error { return ErrTaskReschedule }
