package dbutils

import "github.com/goccy/go-json"

// IsValidJSON checks if the provided string is valid JSON.
func IsValidJSON(data string) bool {
	return json.Valid([]byte(data))
}
