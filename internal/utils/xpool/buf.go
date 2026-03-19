// Package xpool provides a buffer pool for efficient memory reuse.
package xpool

import (
	"bytes"
	"sync"
)

var pool = sync.Pool{New: func() any {
	return new(bytes.Buffer)
}}

// AcquireBuffer retrieves a buffer from the pool or creates a new one if needed.
func AcquireBuffer() *bytes.Buffer {
	val := pool.Get()
	buf, ok := val.(*bytes.Buffer)
	if ok {
		return buf
	}

	return &bytes.Buffer{}
}

// ReleaseBuffer resets and returns a buffer to the pool for reuse.
func ReleaseBuffer(buf *bytes.Buffer) {
	buf.Reset()
	pool.Put(buf)
}
