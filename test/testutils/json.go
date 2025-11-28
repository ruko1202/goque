package testutils

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestPayload represents a simple test payload structure for testing JSON operations.
type TestPayload struct {
	Data string
}

// ToJSON converts an object to its JSON string representation for testing.
func ToJSON(t *testing.T, obj any) string {
	t.Helper()

	b, err := json.Marshal(obj)
	require.NoError(t, err)

	return string(b)
}

// FromJSON deserializes a JSON string into a TestPayload for testing.
func FromJSON(t *testing.T, j string) *TestPayload {
	t.Helper()

	dest := &TestPayload{}
	err := json.Unmarshal([]byte(j), dest)
	require.NoError(t, err)

	return dest
}
