package entity

// TaskProcessingOperations represents the type of operation being performed on a task.
type TaskProcessingOperations = string

// Task processing operation types for metrics and tracking.
const (
	OperationFetch      TaskProcessingOperations = "fetch"      // Task fetching from database
	OperationProcessing TaskProcessingOperations = "processing" // Task execution by processor
	OperationCleanup    TaskProcessingOperations = "cleanup"    // Task cleanup operations
	OperationHealth     TaskProcessingOperations = "health"     // Health check operations
)
