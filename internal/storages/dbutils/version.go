package dbutils

import (
	"database/sql"
	"fmt"

	"github.com/ruko1202/goque/internal/entity"
)

// ExpectRowsAffected checks the result of a version-fenced write. It returns
// entity.ErrTaskVersionConflict when fewer rows than expected were matched,
// i.e. some row's version changed (or the row was deleted) since it was read.
func ExpectRowsAffected(res sql.Result, expected int64, target string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected != expected {
		return fmt.Errorf("%w: %s: matched %d of %d rows", entity.ErrTaskVersionConflict, target, affected, expected)
	}

	return nil
}
