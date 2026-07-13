// Package respwriter provides a pooled http.ResponseWriter wrapper shared by
// the tracing and metric middleware to capture the response status code and
// number of bytes written.
package respwriter

import (
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
)

// Writer wraps an http.ResponseWriter (via httpsnoop) to record the status
// code and number of bytes written to the response body. It is pooled;
// acquire one with Get and return it with Put (typically via defer) once
// the request has finished.
type Writer struct {
	// ResponseWriter is the wrapped writer — pass this, not the original,
	// to the next handler in the chain.
	ResponseWriter http.ResponseWriter
	// StatusCode is the status written (http.StatusOK if WriteHeader was
	// never called explicitly, matching net/http's own default).
	StatusCode int
	// BytesWritten is the cumulative byte count across all Write calls.
	BytesWritten int64

	written bool
}

var pool = &sync.Pool{
	New: func() interface{} { return &Writer{} },
}

// Get returns a pooled *Writer wrapping w.
func Get(w http.ResponseWriter) *Writer {
	rw := pool.Get().(*Writer)
	rw.written = false
	rw.StatusCode = http.StatusOK
	rw.BytesWritten = 0
	rw.ResponseWriter = httpsnoop.Wrap(w, httpsnoop.Hooks{
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				n, err := next(b)
				rw.BytesWritten += int64(n)
				rw.written = true
				return n, err
			}
		},
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(statusCode int) {
				if !rw.written {
					rw.written = true
					rw.StatusCode = statusCode
					// only call next WriteHeader when header is not written yet
					// this is to prevent superfluous WriteHeader call
					next(statusCode)
				}
			}
		},
	})
	return rw
}

// Put resets and returns rw to the pool. Do not use rw after calling Put.
func Put(rw *Writer) {
	rw.ResponseWriter = nil
	pool.Put(rw)
}
