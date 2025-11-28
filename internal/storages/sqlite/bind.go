package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
)

const (
	timeFormat = time.RFC3339
)

func toDBModel(task *entity.Task) *model.Task {
	var updatedAt *string
	if task.UpdatedAt != nil {
		updatedAt = lo.ToPtr(timeToString(lo.FromPtr(task.UpdatedAt)))
	}
	return &model.Task{
		ID:            lo.ToPtr(task.ID.String()),
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		CreatedAt:     timeToString(task.CreatedAt),
		UpdatedAt:     updatedAt,
		NextAttemptAt: timeToString(task.NextAttemptAt),
	}
}

func fromDBModel(task *model.Task) (*entity.Task, error) {
	id, err := uuid.Parse(lo.FromPtr(task.ID))
	if err != nil {
		return nil, fmt.Errorf("parse task id: %w", err)
	}
	var updatedAt *time.Time
	if task.UpdatedAt != nil {
		updatedAt = lo.ToPtr(timeFromString(lo.FromPtr(task.UpdatedAt)))
	}

	return &entity.Task{
		ID:            id,
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		CreatedAt:     timeFromString(task.CreatedAt),
		UpdatedAt:     updatedAt,
		NextAttemptAt: timeFromString(task.NextAttemptAt),
	}, nil
}

func fromDBModels(dbTasks []*model.Task) ([]*entity.Task, error) {
	tasks := make([]*entity.Task, 0, len(dbTasks))
	for _, dbTask := range dbTasks {
		task, err := fromDBModel(dbTask)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func timeToString(t time.Time) string {
	return t.Format(timeFormat)
}

func timeFromString(value string) time.Time {
	t, err := time.Parse(timeFormat, value)
	if err != nil {
		xlog.Error(context.Background(), "parse time error", zap.Error(err), zap.String("time", value))
		return time.Time{}
	}

	return t
}
