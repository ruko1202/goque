package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
)

const (
	timeFormat = time.RFC3339
)

func toDBModel(ctx context.Context, task *entity.Task) *model.GoqueTask {
	metadata := task.Metadata.Merge(goquectx.Values(ctx))
	var updatedAt *string
	if task.UpdatedAt != nil {
		updatedAt = lo.ToPtr(timeToString(lo.FromPtr(task.UpdatedAt)))
	}
	return &model.GoqueTask{
		ID:            lo.ToPtr(task.ID.String()),
		Type:          task.Type,
		ExternalID:    task.ExternalID,
		Payload:       task.Payload,
		Status:        task.Status,
		Attempts:      task.Attempts,
		Errors:        task.Errors,
		Metadata:      lo.ToPtr(metadata.ToJSON(ctx)),
		CreatedAt:     timeToString(task.CreatedAt),
		UpdatedAt:     updatedAt,
		NextAttemptAt: timeToString(task.NextAttemptAt),
	}
}

func fromDBModel(ctx context.Context, task *model.GoqueTask) (*entity.Task, error) {
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
		Metadata:      entity.NewMetadataFromJSON(ctx, task.Metadata),
		CreatedAt:     timeFromString(task.CreatedAt),
		UpdatedAt:     updatedAt,
		NextAttemptAt: timeFromString(task.NextAttemptAt),
	}, nil
}

func fromDBModels(ctx context.Context, dbTasks []*model.GoqueTask) ([]*entity.Task, error) {
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

// timeToString serializes t to RFC3339 in UTC. Forcing UTC is what
// keeps lexicographic compare consistent with chronological order:
// SQLite stores TEXT, and a row written with a local-zone offset
// (e.g. "2026-05-25T16:00:00+03:00") would compare wrong against a
// UTC-zone row ("2026-05-25T13:00:00Z") even though both moments
// are equal. All WHERE clauses in this package format the comparison
// value via timeToString too, so as long as everything goes through
// here the order is stable. See RUK-139 for the underlying hazard
// and why we did NOT migrate the column to INTEGER unix-seconds:
// SQLite is positioned as dev/test only and the UTC normalisation
// closes the practical risk surface.
func timeToString(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func timeFromString(value string) time.Time {
	t, err := time.Parse(timeFormat, value)
	if err != nil {
		xlog.Error(context.Background(), "parse time error", xfield.Error(err), xfield.String("time", value))
		return time.Time{}
	}

	return t
}
