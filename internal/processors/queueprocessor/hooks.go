package queueprocessor

import (
	"context"
	"reflect"
	"runtime"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
)

type (
	// HookBeforeProcessing defines a hook function called before task processing begins.
	HookBeforeProcessing func(ctx context.Context, task *entity.Task)
	// HookAfterProcessing defines a hook function called after task processing completes.
	HookAfterProcessing func(ctx context.Context, task *entity.Task, err error)
)

// LoggingBeforeProcessing default log before processing the task.
func LoggingBeforeProcessing(ctx context.Context, task *entity.Task) {
	xlog.Info(ctx, "processing task",
		xfield.String("externalID", task.ExternalID),
		xfield.String("type", task.Type),
		xfield.String("status", task.Status),
		xfield.String("errors", lo.FromPtr(task.Errors)),
		xfield.Time("createdAt", task.CreatedAt),
		xfield.Any("updatedAt", task.UpdatedAt),
	)
}

// LoggingAfterProcessing default log after processing the task.
func LoggingAfterProcessing(ctx context.Context, task *entity.Task, err error) {
	if err != nil {
		xlog.Error(ctx, "failed to process task",
			xfield.String("externalID", task.ExternalID),
			xfield.String("type", task.Type),
			xfield.String("status", task.Status),
			xfield.String("errors", lo.FromPtr(task.Errors)),
			xfield.Time("createdAt", task.CreatedAt),
			xfield.Any("updatedAt", task.UpdatedAt),
			xfield.Error(err),
		)
		return
	}

	xlog.Info(ctx, "process task successfully")
}

func getHookFuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
