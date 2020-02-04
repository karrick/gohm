package gohm

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// responseWriter must behave exactly like http.ResponseWriter, yet store up
// response until query complete.
//
// Only need locks on Header, Write, and WriteHeader, so if this is in the
// middle of dealing with a handler timeout, and the handler invokes one of
// these methods, we prevent race case.
type responseWriter struct {
	// size 24
	begin, end time.Time // begin and end track the duration of the request for logging purposes

	// size 16
	requestHeaders map[string]string
	responseError  string
	responseWriter http.ResponseWriter

	// size 8
	bytesWritten    int64
	responseBody    *bytes.Buffer
	responseHeaders http.Header
	responseStatus  int

	// size 4 or 8
	lock sync.Mutex

	// size 1
	timedOut    bool
	wroteHeader bool
}

func (rw *responseWriter) handlerComplete() {
	if rw.responseHeaders != nil {
		responseHeaders := rw.responseWriter.Header()
		for k, vv := range rw.responseHeaders {
			responseHeaders[k] = vv
		}
	}

	if !rw.wroteHeader {
		rw.writeHeader(http.StatusOK)
	}
	rw.responseWriter.WriteHeader(rw.responseStatus)

	n, err := rw.responseWriter.Write(rw.responseBody.Bytes()) // buffer.WriteTo would Reset after dumping its contents, but that is not needed
	if err != nil {
		// We want our logs to reflect when we cannot write the results to the
		// underlying http.ResponseWriter.  NOTE: This does not change what was
		// actually written to the client, because that has already failed.
		rw.responseError = err.Error()
		rw.responseStatus = http.StatusInternalServerError
	}

	// for access log
	rw.bytesWritten = int64(n)
	rw.end = time.Now()
}

func (rw *responseWriter) handlerError(error string, status int) {
	// Defer to standard library when there was a handler error.
	http.Error(rw.responseWriter, error, status)

	// Save parameter values for log.
	rw.bytesWritten = int64(len(error) + 1) // account for appended newline
	rw.end = time.Now()
	rw.responseError = error
	rw.writeHeader(status)
}

func (rw *responseWriter) handlerTimeout(error string, status int) {
	rw.lock.Lock()
	rw.handlerError(error, status)
	rw.timedOut = true // When handler attempts to Write it will receive appropriate error.
	rw.lock.Unlock()
}

// Header returns the header map that will be sent by WriteHeader.
func (rw *responseWriter) Header() http.Header {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	if rw.responseHeaders == nil {
		rw.responseHeaders = make(http.Header)
	}
	return rw.responseHeaders
}

// Write writes the data to the connection as part of an HTTP reply.
func (rw *responseWriter) Write(blob []byte) (int, error) {
	rw.lock.Lock()
	if rw.timedOut {
		rw.lock.Unlock()
		return 0, http.ErrHandlerTimeout // inform handler it is beyond timeout
	}
	if !rw.wroteHeader {
		rw.writeHeader(http.StatusOK)
	}
	n, err := rw.responseBody.Write(blob)
	rw.lock.Unlock()
	return n, err
}

// WriteHeader sends an HTTP response header with the provided status code.
func (rw *responseWriter) WriteHeader(status int) {
	rw.lock.Lock()
	if !(rw.timedOut || rw.wroteHeader) {
		rw.writeHeader(status)
	}
	rw.lock.Unlock()
}

func (rw *responseWriter) writeHeader(status int) {
	rw.wroteHeader = true
	rw.responseStatus = status
}
