package task

import (
	"context"

	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/postgres/public/model"
	"github.com/ruko1202/goque/internal/utils/goquectx"
)

func toDBModel(ctx context.Context, task *entity.Task) *model.GoqueTask {
	metadata := task.Metadata.Merge(goquectx.Values(ctx))
	return &model.GoqueTask{
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

func fromDBModel(ctx context.Context, task *model.GoqueTask) *entity.Task {
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

func fromDBModels(ctx context.Context, tasks []*model.GoqueTask) []*entity.Task {
	return lo.Map(tasks, func(item *model.GoqueTask, _ int) *entity.Task {
		return fromDBModel(ctx, item)
	})
}
