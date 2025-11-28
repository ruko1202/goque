package processor

import "fmt"

var (
	// ErrTaskCancel is returned when a task is canceled during processing.
	ErrTaskCancel = fmt.Errorf("task canceled")
	// ErrTaskTimeout is returned when task processing exceeds the timeout limit.
	ErrTaskTimeout = fmt.Errorf("task processing timeout")
)
