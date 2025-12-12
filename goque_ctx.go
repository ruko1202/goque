package goque

import (
	"github.com/ruko1202/goque/internal/utils/goquectx"
)

// Context value functions for storing and retrieving task metadata.
var (
	// WithValue adds a single key-value pair to the context for task metadata tracking.
	WithValue = goquectx.WithValue
	// WithValues adds multiple key-value pairs to the context for task metadata tracking.
	WithValues = goquectx.WithValues
	// ValueByKey retrieves stored value from the context by key.
	ValueByKey = goquectx.ValueByKey
	// Values retrieves all stored metadata values from the context.
	Values = goquectx.Values
)
