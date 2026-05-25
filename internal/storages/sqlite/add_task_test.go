package sqlite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

// TestHandleError_UniqueConstraint protects against silent regressions
// in the sqlite duplicate-detection path. We string-match instead of
// type-asserting on sqlite3.Error to avoid forcing CGO on downstream
// consumers (PG/MySQL-only services ship as scratch/distroless and
// can't link mattn/go-sqlite3). The integration test in
// internal/storages/test/add_task_test.go exercises the real driver
// end-to-end; this test pins the substring contract of handleError.
func TestHandleError_UniqueConstraint(t *testing.T) {
	t.Parallel()

	t.Run("nil passes through", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, handleError(nil))
	})

	t.Run("unique constraint message maps to ErrDuplicateTask", func(t *testing.T) {
		t.Parallel()
		// Real message from mattn/go-sqlite3 v1.14.x:
		// "UNIQUE constraint failed: goque_task.type, goque_task.external_id"
		src := errors.New("UNIQUE constraint failed: goque_task.type, goque_task.external_id")
		require.ErrorIs(t, handleError(src), entity.ErrDuplicateTask)
	})

	t.Run("unique constraint with future driver decorations", func(t *testing.T) {
		t.Parallel()
		// A defensive test: if the driver ever prefixes/suffixes the
		// message we still want to fire as long as the canonical
		// "UNIQUE constraint failed" substring is present.
		src := errors.New("sqlite3: (1555) UNIQUE constraint failed: goque_task.type, goque_task.external_id")
		require.ErrorIs(t, handleError(src), entity.ErrDuplicateTask)
	})

	t.Run("other constraint passes through unchanged", func(t *testing.T) {
		t.Parallel()
		src := errors.New("NOT NULL constraint failed: goque_task.type")
		require.NotErrorIs(t, handleError(src), entity.ErrDuplicateTask)
		require.Equal(t, src, handleError(src))
	})

	t.Run("non-sqlite error passes through unchanged", func(t *testing.T) {
		t.Parallel()
		src := errors.New("some random error")
		require.NotErrorIs(t, handleError(src), entity.ErrDuplicateTask)
		require.Equal(t, src, handleError(src))
	})
}
