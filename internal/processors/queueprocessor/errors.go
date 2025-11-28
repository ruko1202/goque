package queueprocessor

import (
	"errors"
)

var (
	// ErrTaskCancel is returned when a task is canceled during processing.
	ErrTaskCancel = errors.New("task canceled")
	// ErrTaskTimeout is returned when task processing exceeds the timeout limit.
	ErrTaskTimeout = errors.New("task processing timeout")
)
