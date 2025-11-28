// Package metrics provides Prometheus instrumentation for Goque task queue operations.
// It includes counters, gauges, and histograms for tracking task processing, worker pools,
// retry attempts, payload sizes, and operation durations.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ruko1202/goque/internal/entity"
)

const promSubsystem = "goque"

const (
	labelStatus                   = "status"
	labelTaskType                 = "task_type"
	labelTaskProcessingOperations = "operation"
)

var (
	processedTasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "processed_tasks_total",
			Help:        "Total number of processed tasks by task type and status",
			ConstLabels: constLabels,
		},
		[]string{labelTaskType, labelStatus},
	)

	taskProcessingDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "task_processing_duration_seconds",
			Help:        "Task processing duration in seconds by task type",
			Buckets:     []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
			ConstLabels: constLabels,
		},
		[]string{labelTaskType, labelTaskProcessingOperations},
	)
	taskRetryAttempts = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "task_retry_attempts",
			Help:        "Number of retry attempts per task by task type",
			ConstLabels: constLabels,
			Buckets:     []float64{1, 2, 3, 5, 10, 15, 20, 30, 50},
		},
		[]string{labelTaskType},
	)
	taskPayloadSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "task_payload_size_bytes",
			Help:        "Size of user-provided payload in bytes.",
			ConstLabels: constLabels,
			// 100B, 1KB, 10KB, 100KB, 1MB, 5MB
			Buckets: []float64{100, 1_000, 10_000, 100_000, 1_000_000, 5_000_000},
		},
		[]string{labelTaskType},
	)
	processorsWorkersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "processors_workers_count",
			Help:        "Current number of processors workers by task type",
			ConstLabels: constLabels,
		},
		[]string{labelTaskType},
	)
	operationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   promSubsystem,
			Name:        "operations_total",
			Help:        "Total number of operations by task type",
			ConstLabels: constLabels,
		},
		[]string{labelTaskType, labelTaskProcessingOperations},
	)
)

// IncProcessingTasks increments the counter of processed tasks for the given task type and status.
func IncProcessingTasks(taskType entity.TaskType, status entity.TaskStatus) {
	processedTasksTotal.With(prometheus.Labels{
		labelTaskType: taskType,
		labelStatus:   status,
	}).Inc()
}

// SetTaskRetryAttempts records the number of retry attempts for a task.
func SetTaskRetryAttempts(taskType entity.TaskType, retryAttempts int32) {
	taskRetryAttempts.With(prometheus.Labels{
		labelTaskType: taskType,
	}).Observe(float64(retryAttempts))
}

// SetTaskPayloadSize records the size of the task payload in bytes.
func SetTaskPayloadSize(taskType entity.TaskType, size int) {
	taskPayloadSize.With(prometheus.Labels{
		labelTaskType: taskType,
	}).Observe(float64(size))
}

// TaskProcessingDurationSecondsObserver returns an observer for recording task processing duration.
func TaskProcessingDurationSecondsObserver(taskType entity.TaskType, operations entity.TaskProcessingOperations) prometheus.Observer {
	return taskProcessingDurationSeconds.With(prometheus.Labels{
		labelTaskType:                 taskType,
		labelTaskProcessingOperations: operations,
	})
}

// SetTasksWorkersTotal sets the current number of active workers for a task type.
func SetTasksWorkersTotal(taskType entity.TaskType, count int) {
	processorsWorkersTotal.With(prometheus.Labels{
		labelTaskType: taskType,
	}).Set(float64(count))
}

// SetOperationsTotal adds to the counter of operations performed for a task type.
func SetOperationsTotal(taskType entity.TaskType, operations entity.TaskProcessingOperations, count int) {
	operationsTotal.With(prometheus.Labels{
		labelTaskType:                 taskType,
		labelTaskProcessingOperations: operations,
	}).Add(float64(count))
}
