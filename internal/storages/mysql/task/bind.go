package mysqltask

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
)

func toDBModel(task *entity.Task) *model.Task {
	return &model.Task{
		ID:            task.ID.String(),
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

func fromDBModel(task *model.Task) (*entity.Task, error) {
	id, err := uuid.Parse(task.ID)
	if err != nil {
		return nil, fmt.Errorf("parse task id: %w", err)
	}
	return &entity.Task{
		ID:            id,
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		NextAttemptAt: task.NextAttemptAt,
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
