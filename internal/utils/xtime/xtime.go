// Package xtime provides time-related utility functions.
package xtime

import "time"

// Now returns the current time in UTC.
func Now() time.Time {
	return time.Now().UTC()
}
