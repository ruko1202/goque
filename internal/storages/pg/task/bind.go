package task

import (
	"context"

	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/utils/goquectx"
)

func toDBModel(ctx context.Context, task *entity.Task) *model.Task {
	metadata := task.Metadata.Merge(goquectx.Values(ctx))
	return &model.Task{
		ID:            task.ID,
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

func fromDBModel(ctx context.Context, task *model.Task) *entity.Task {
	return &entity.Task{
		ID:            task.ID,
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
	}
}

func fromDBModels(ctx context.Context, tasks []*model.Task) []*entity.Task {
	return lo.Map(tasks, func(item *model.Task, _ int) *entity.Task {
		return fromDBModel(ctx, item)
	})
}
