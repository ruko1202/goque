package sqlite

import (
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/model"
	"github.com/ruko1202/goque/internal/pkg/generated/sqlite3/table"
)

// versionPredicate matches every task by id and the version it was read with:
// (id = ? AND version = ?) OR (id = ? AND version = ?) ...
//
// SQLite parses a flat OR chain as a left-deep tree and rejects trees deeper
// than 1000, so a full batch (cure takes up to 1000 rows) would fail. The ORs
// are nested as a balanced tree instead, keeping the depth logarithmic.
// tasks must not be empty.
func versionPredicate(tasks []*model.GoqueTask) sqlite.BoolExpression {
	return balancedOR(lo.Map(tasks, func(task *model.GoqueTask, _ int) sqlite.BoolExpression {
		return sqlite.AND(
			table.GoqueTask.ID.EQ(sqlite.String(lo.FromPtr(task.ID))),
			table.GoqueTask.Version.EQ(sqlite.Int(task.Version)),
		)
	}))
}

func balancedOR(exprs []sqlite.BoolExpression) sqlite.BoolExpression {
	if len(exprs) == 1 {
		return exprs[0]
	}
	mid := len(exprs) / 2

	return sqlite.OR(balancedOR(exprs[:mid]), balancedOR(exprs[mid:]))
}
