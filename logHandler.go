package gohm

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const apacheLogFormat = "%s [%s] \"%s\" %d %d %f\n"
const timeFormat = "02/Jan/2006:15:04:05 MST"

// ErrorLogHandler returns a new http.Handler that logs HTTP requests that result in response
// errors. The handler will output lines in common log format to the specified io.Writer.
func ErrorLogHandler(out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggedResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		begin := time.Now()
		next.ServeHTTP(lrw, r)

		// NOTE: check for status zero value because when omitted by handler, it's filled in later in http stack
		if lrw.status != http.StatusOK && lrw.status != 0 {
			end := time.Now()
			clientIP := r.RemoteAddr
			if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
				clientIP = clientIP[:colon]
			}
			request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
			duration := end.Sub(begin).Seconds()
			formattedTime := end.UTC().Format(timeFormat)
			fmt.Fprintf(out, apacheLogFormat, clientIP, formattedTime, request, lrw.status, lrw.responseBytes, duration)
		}
	})
}

// LogHandler returns a new http.Handler that logs HTTP requests and responses in common log format
// to the specified io.Writer.
func LogHandler(out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggedResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		begin := time.Now()
		next.ServeHTTP(lrw, r)
		end := time.Now()

		clientIP := r.RemoteAddr
		if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
			clientIP = clientIP[:colon]
		}
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)

		duration := end.Sub(begin).Seconds()
		formattedTime := end.UTC().Format(timeFormat)
		fmt.Fprintf(out, apacheLogFormat, clientIP, formattedTime, request, lrw.status, lrw.responseBytes, duration)
	})
}

type loggedResponseWriter struct {
	http.ResponseWriter
	responseBytes int64
	status        int
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
