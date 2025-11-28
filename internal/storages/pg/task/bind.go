package task

import (
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
)

func toDBModel(task *entity.Task) *model.Task {
	return &model.Task{
		ID:            task.ID,
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		NextAttemptAt: task.NextAttemptAt,
	}
}

func fromDBModel(task *model.Task) *entity.Task {
	return &entity.Task{
		ID:            task.ID,
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		NextAttemptAt: task.NextAttemptAt,
	}
}

func fromDBModels(tasks []*model.Task) []*entity.Task {
	return lo.Map(tasks, func(item *model.Task, _ int) *entity.Task {
		return fromDBModel(item)
	})
}
