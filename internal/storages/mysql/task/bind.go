package mysqltask

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/mysql/goque/model"
)

func toDBModel(ctx context.Context, task *entity.Task) *model.Task {
	metadata := task.Metadata.Merge(goquectx.ValuesFromContext(ctx))
	return &model.Task{
		ID:            task.ID.String(),
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		Metadata:      lo.ToPtr(metadata.ToJSON(ctx)),
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		NextAttemptAt: task.NextAttemptAt,
	}
}

func fromDBModel(ctx context.Context, task *model.Task) (*entity.Task, error) {
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
		Metadata:      entity.NewMetadataFromJSON(ctx, task.Metadata),
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		NextAttemptAt: task.NextAttemptAt,
	}, nil
}

func fromDBModels(ctx context.Context, dbTasks []*model.Task) ([]*entity.Task, error) {
	tasks := make([]*entity.Task, 0, len(dbTasks))
	for _, dbTask := range dbTasks {
		task, err := fromDBModel(ctx, dbTask)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
