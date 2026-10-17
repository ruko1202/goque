package sqlite

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/go-jet/jet/v2/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
)

// TestVersionPredicate_FullBatch pins that the per-row version predicate works
// for a full cure batch (1000 rows): a flat OR chain of that size exceeds
// SQLite's expression depth limit ("Expression tree is too large").
func TestVersionPredicate_FullBatch(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE goque_task (id TEXT PRIMARY KEY, version INTEGER NOT NULL DEFAULT 0)`)
	require.NoError(t, err)

	const count = 1000
	tasks := make([]*model.GoqueTask, 0, count)
	for i := range count {
		id := fmt.Sprintf("task-%d", i)
		_, err = db.Exec(`INSERT INTO goque_task (id, version) VALUES (?, 3)`, id)
		require.NoError(t, err)
		tasks = append(tasks, &model.GoqueTask{ID: lo.ToPtr(id), Version: 3})
	}
	// A stale version on one row must leave that row out of the match.
	tasks[count/2].Version = 2

	query, args := table.GoqueTask.
		UPDATE(table.GoqueTask.Version).
		SET(table.GoqueTask.Version.ADD(sqlite.Int(1))).
		WHERE(versionPredicate(tasks)).
		Sql()
	res, err := db.Exec(query, args...)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(count-1), affected)
}
