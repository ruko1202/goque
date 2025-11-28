// Package dbutils provides common database utilities for task storage implementations.
package dbutils

import (
	"github.com/go-jet/jet/v2/mysql"
	"github.com/go-jet/jet/v2/postgres"
)

// PgWhereBuilder constructs PostgreSQL WHERE clauses using AND logic.
type PgWhereBuilder struct {
	expr postgres.BoolExpression
}

// NewPgWhereBuilder creates a new PostgreSQL WHERE clause builder.
func NewPgWhereBuilder() *PgWhereBuilder {
	return &PgWhereBuilder{}
}

// And adds a boolean expression to the WHERE clause using AND logic.
func (w *PgWhereBuilder) And(expr postgres.BoolExpression) {
	if w.expr == nil {
		w.expr = expr
		return
	}

	w.expr = w.expr.AND(expr)
}

// Expression returns the final constructed boolean expression.
func (w *PgWhereBuilder) Expression() postgres.BoolExpression {
	return w.expr
}

// MysqlWhereBuilder constructs MySQL WHERE clauses using AND logic.
type MysqlWhereBuilder struct {
	expr postgres.BoolExpression
}

// NewMysqlWhereBuilder creates a new MySQL WHERE clause builder.
func NewMysqlWhereBuilder() *MysqlWhereBuilder {
	return &MysqlWhereBuilder{}
}

// And adds a boolean expression to the WHERE clause using AND logic.
func (w *MysqlWhereBuilder) And(expr mysql.BoolExpression) {
	if w.expr == nil {
		w.expr = expr
		return
	}

	w.expr = w.expr.AND(expr)
}

// Expression returns the final constructed boolean expression.
func (w *MysqlWhereBuilder) Expression() mysql.BoolExpression {
	return w.expr
}
