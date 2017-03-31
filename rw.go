package gohm

import (
	"bytes"
	"fmt"
	"net/http"
)

// responseWriter must behave exactly like http.responseWriter, yet store up response until after
// complete. maybe we provide Flush method to trigger completion.
type responseWriter struct {
	wrapped http.ResponseWriter
	header  http.Header
	buf     bytes.Buffer
	status  int
}

func ResponseWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			wrapped: w,
			header:  make(http.Header),
		}

		next.ServeHTTP(rw, r)

		if err := rw.flush(); err != nil {
			Error(rw, fmt.Sprintf("cannot flush response writer: %s", err), http.StatusInternalServerError)
		}
	})
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) Write(blob []byte) (int, error) {
	return rw.buf.Write(blob)
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
}

func (rw *responseWriter) flush() error {
	// write header
	header := rw.wrapped.Header()
	for key, values := range rw.header {
		for _, value := range values {
			header.Add(key, value)
		}
	}

	// write status
	rw.wrapped.WriteHeader(rw.status)

	// write response
	_, err := rw.buf.WriteTo(rw.wrapped)
	return err
}
