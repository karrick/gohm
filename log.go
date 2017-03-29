package gohm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const apacheLogFormat = "%s [%s] \"%s\" %d %d %f\n"
const apacheTimeFormat = "02/Jan/2006:15:04:05 MST"

const (
	LogStatus1xx uint32 = 1  // LogStatus1xx used to log HTTP requests which have a 1xx response
	LogStatus2xx uint32 = 2  // LogStatus2xx used to log HTTP requests which have a 2xx response
	LogStatus3xx uint32 = 4  // LogStatus3xx used to log HTTP requests which have a 3xx response
	LogStatus4xx uint32 = 8  // LogStatus4xx used to log HTTP requests which have a 4xx response
	LogStatus5xx uint32 = 16 // LogStatus5xx used to log HTTP requests which have a 5xx response
)

type loggedResponseWriter struct {
	http.ResponseWriter
	responseBytes int64
	status        int
	begin, end    time.Time
}

func (r *loggedResponseWriter) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.responseBytes += int64(written)
	return written, err
}

func (r *loggedResponseWriter) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func clf(r *http.Request, lrw *loggedResponseWriter, out io.Writer) {
	clientIP := r.RemoteAddr
	if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
		clientIP = clientIP[:colon]
	}
	request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
	duration := lrw.end.Sub(lrw.begin).Seconds()
	formattedTime := lrw.end.UTC().Format(apacheTimeFormat)

	line := []byte(fmt.Sprintf(apacheLogFormat, clientIP, formattedTime, request, lrw.status, lrw.responseBytes, duration))
	if _, err := out.Write(line); err != nil {
		// if we cannot write to out for some reason, write it to stderr
		_, _ = os.Stderr.Write(line)
	}
}

// LogAll returns a new http.Handler that logs HTTP requests and responses in common log format
// to the specified io.Writer.
//
// 	mux := http.NewServeMux()
// 	mux.Handle("/example/path", gohm.LogAll(os.Stderr, decodeURI(expand(querier))))
func LogAll(out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggedResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			begin:          time.Now(),
		}

		next.ServeHTTP(lrw, r)

		lrw.end = time.Now()
		clf(r, lrw, out)
	})
}

// LogErrors returns a new http.Handler that logs HTTP requests that result in response errors, or
// more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output lines
// in common log format to the specified io.Writer.
//
// 	mux := http.NewServeMux()
// 	mux.Handle("/example/path", gohm.LogErrors(os.Stderr, decodeURI(expand(querier))))
func LogErrors(out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggedResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			begin:          time.Now(),
		}

		next.ServeHTTP(lrw, r)

		if group := lrw.status / 100; group == 4 || group == 5 {
			lrw.end = time.Now()
			clf(r, lrw, out)
		}
	})
}

// LogStatusBitmask returns a new http.Handler that logs HTTP requests that have a status code that
// matches any of the status codes in the specified bitmask.  The handler will output lines in
// common log format to the specified io.Writer.
//
// The closed over `bitmask` parameter is used to specify which HTTP requests ought to be logged
// based on the service's HTTP status code for each request.
//
// The default log format line is:
//
//      "{client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//
// 	mux := http.NewServeMux()
// 	logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
// 	mux.Handle("/example/path", gohm.LogStatusBitmask(&logBitmask, os.Stderr, decodeURI(expand(querier))))
func LogStatusBitmask(bitmask *uint32, out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggedResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			begin:          time.Now(),
		}

		next.ServeHTTP(lrw, r)

		var bit uint32
		switch lrw.status / 100 {
		case 1:
			bit = 1
		case 2:
			bit = 2
		case 3:
			bit = 4
		case 4:
			bit = 8
		case 5:
			bit = 16
		}

		if bit == 0 || (atomic.LoadUint32(bitmask))&bit > 0 {
			lrw.end = time.Now()
			clf(r, lrw, out)
		}
	})
}
