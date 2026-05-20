// Package goque provides a robust, SQL-backed task queue system for Go applications.
package goque

import (
	"github.com/ruko1202/goque/internal/entity"
)

var (
	// ErrDuplicateTask is returned when attempting to insert a task with a duplicate external ID.
	ErrDuplicateTask = entity.ErrDuplicateTask
	// ErrInvalidPayloadFormat is returned when the task payload is not valid JSON.
	ErrInvalidPayloadFormat = entity.ErrInvalidPayloadFormat
	// ErrPayloadMarshal is returned when a typed task payload cannot be marshaled to JSON.
	ErrPayloadMarshal = entity.ErrPayloadMarshal
	// ErrPayloadUnmarshal is returned when a typed task payload cannot be unmarshaled from JSON.
	ErrPayloadUnmarshal = entity.ErrPayloadUnmarshal
	// ErrTaskCancel is returned when a task is canceled during processing.
	ErrTaskCancel = entity.ErrTaskCancel
	// ErrTaskTimeout is returned when task processing exceeds the timeout limit.
	ErrTaskTimeout = entity.ErrTaskTimeout
)
