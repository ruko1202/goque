package sqlite

import (
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

// TestHandleError_UniqueConstraint protects against silent regressions in
// the sqlite duplicate-detection path. Prior implementations matched the
// raw error message string — a table rename or driver upgrade would
// silently break it. The current implementation must rely on the typed
// error code (sqlite3.ErrConstraintUnique).
func TestHandleError_UniqueConstraint(t *testing.T) {
	t.Parallel()

	t.Run("nil passes through", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, handleError(nil))
	})

	t.Run("unique constraint maps to ErrDuplicateTask", func(t *testing.T) {
		t.Parallel()
		src := sqlite3.Error{
			Code:         sqlite3.ErrConstraint,
			ExtendedCode: sqlite3.ErrConstraintUnique,
		}
		require.ErrorIs(t, handleError(src), entity.ErrDuplicateTask)
	})

	t.Run("other constraint passes through unchanged", func(t *testing.T) {
		t.Parallel()
		src := sqlite3.Error{
			Code:         sqlite3.ErrConstraint,
			ExtendedCode: sqlite3.ErrConstraintNotNull,
		}
		got := handleError(src)
		require.NotErrorIs(t, got, entity.ErrDuplicateTask)
		require.Error(t, got)
	})

	t.Run("non-sqlite error passes through unchanged", func(t *testing.T) {
		t.Parallel()
		src := errors.New("some random error")
		got := handleError(src)
		require.NotErrorIs(t, got, entity.ErrDuplicateTask)
		require.Equal(t, src, got)
	})
}
