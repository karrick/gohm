package gohm

import (
	"bytes"
	"net/http"
)

// responseWriter must behave exactly like http.responseWriter, yet store up response until after
// complete. maybe we provide Flush method to trigger completion.
type responseWriter struct {
	http.ResponseWriter
	header        http.Header
	body          bytes.Buffer
	status        int
	closeNotifyCh <-chan bool
	statusWritten bool
}

func (rw *responseWriter) Header() http.Header {
	m := rw.header
	if m == nil {
		m = make(http.Header)
		rw.header = m
	}
	return m
}

func (rw *responseWriter) Write(blob []byte) (int, error) {
	return rw.body.Write(blob)
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.statusWritten = true
}

func (rw *responseWriter) flush() error {
	// write header
	header := rw.ResponseWriter.Header()
	for key, values := range rw.header {
		for _, value := range values {
			header.Add(key, value)
		}
	}

	// write status
	if !rw.statusWritten {
		rw.status = http.StatusOK
	}
	rw.ResponseWriter.WriteHeader(rw.status)

	// write response
	_, err := rw.body.WriteTo(rw.ResponseWriter)
	return err
}
