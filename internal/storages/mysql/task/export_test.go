package mysqltask

// Test hooks for mysqltask_test.
var (
	WithinTx                 = (*Storage).withinTx
	SelectTasksForProcessing = (*Storage).getTasksForProcessing
)
