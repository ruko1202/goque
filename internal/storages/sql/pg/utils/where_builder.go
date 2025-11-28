// Package sqlpgutils provides database utility functions and helpers.
package sqlpgutils

import "github.com/go-jet/jet/v2/postgres"

// WhereBuilder helps construct complex WHERE clauses by chaining AND conditions.
type WhereBuilder struct {
	expr postgres.BoolExpression
}

// NewWhereBuilder creates a new empty WHERE clause builder.
func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{}
}

// And adds a boolean expression to the WHERE clause using AND logic.
func (w *WhereBuilder) And(expr postgres.BoolExpression) {
	if w.expr == nil {
		w.expr = expr
		return
	}

	w.expr = w.expr.AND(expr)
}

// Expression returns the final constructed boolean expression.
func (w *WhereBuilder) Expression() postgres.BoolExpression {
	return w.expr
}
