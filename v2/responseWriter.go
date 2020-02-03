package gohm

import (
	"bytes"
	"net/http"
	"time"
)

// responseWriter must behave exactly like http.ResponseWriter, yet store up
// response until query complete.
type responseWriter struct {
	// size 24
	begin, end time.Time // begin and end track the duration of the request for logging purposes

	// size 16
	error                string
	hrw                  http.ResponseWriter
	loggedRequestHeaders map[string]string

	// size 8
	body         *bytes.Buffer
	header       http.Header
	bytesWritten int64
	status       int
}

func (grw *responseWriter) handlerComplete() {
	if grw.header != nil {
		h := grw.hrw.Header()
		for k, vv := range grw.header {
			for _, v := range vv {
				h.Add(k, v)
			}
		}
	}

	if grw.status == 0 {
		grw.status = http.StatusOK // for log
	} else {
		grw.hrw.WriteHeader(grw.status)
	}

	n, err := grw.hrw.Write(grw.body.Bytes()) // buffer.WriteTo would Reset after dumping its contents, but that is not needed

	if err != nil {
		// We want our logs to reflect when we cannot write the results to the
		// underlying http.ResponseWriter.  NOTE: This does not change what was
		// actually written to the client, because that has already failed.
		grw.error = err.Error()
		grw.status = http.StatusInternalServerError
	}

	// for log
	grw.bytesWritten = int64(n)
	grw.end = time.Now()
}

func (grw *responseWriter) handlerError(error string, code int) {
	// Defer to standard library when there was a handler error.
	http.Error(grw.hrw, error, code)

	// Save parameter values for log.
	grw.end = time.Now()
	grw.error = error
	grw.status = code
}

// Header returns the header map that will be sent by WriteHeader.
func (grw *responseWriter) Header() http.Header {
	if grw.header == nil {
		grw.header = make(http.Header)
	}
	return grw.header
}

// Write writes the data to the connection as part of an HTTP reply.
func (grw *responseWriter) Write(blob []byte) (int, error) {
	return grw.body.Write(blob)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (grw *responseWriter) WriteHeader(status int) {
	grw.status = status
}
