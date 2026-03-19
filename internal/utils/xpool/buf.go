package xpool

import (
	"bytes"
	"sync"
)

var pool = sync.Pool{New: func() any {
	return new(bytes.Buffer)
}}

func AcquireBuffer() *bytes.Buffer {
	val := pool.Get()
	buf, ok := val.(*bytes.Buffer)
	if ok {
		return buf
	}

	return &bytes.Buffer{}
}

func ReleaseBuffer(buf *bytes.Buffer) {
	buf.Reset()
	pool.Put(buf)
}
