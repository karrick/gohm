package gohm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	fmt.Fprintf(out, apacheLogFormat, clientIP, formattedTime, request, lrw.status, lrw.responseBytes, duration)
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
		bm := atomic.LoadUint32(bitmask)

		if bit == 0 || bm&bit > 0 {
			lrw.end = time.Now()
			clf(r, lrw, out)
		}
	})
}

// LogStatusBitmaskWithFormat returns a new http.Handler that logs HTTP requests that have a status
// code that matches any of the status codes in the specified bitmask.  The handler will output
// lines in the specified log format string to the specified io.Writer.
//
// The following format directives are supported:
//
// 	begin:           time request received (apache log time format)
// 	begin-epoch:     time request received (epoch)
// 	begin-iso8601:   time request received (ISO-8601 time format)
// 	bytes:           response size
// 	client:          client-ip:client-port
// 	client-ip:       client IP address
// 	client-port:     client port
// 	duration:        duration of request from begin to end, (seconds with millisecond precision)
// 	end:             time request completed (apache log time format)
// 	end-epoch:       time request completed (epoch)
// 	end-iso8601:     time request completed (ISO-8601 time format)
// 	method:          request method, e.g., GET or POST
// 	proto:           request protocol, e.g., HTTP/1.1
// 	status:          response status code
// 	uri:             request URI
//
// In addition, values from HTTP request headers can also be included in the log by prefixing the
// HTTP header name with "http-", as shown below:
//
//      format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//
// The closed over `bitmask` parameter is used to specify which HTTP requests ought to be logged
// based on the service's HTTP status code for each request.
//
// 	mux := http.NewServeMux()
// 	logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
// 	mux.Handle("/example/path", gohm.LogStatusBitmask(&logBitmask, os.Stderr, decodeURI(expand(querier))))
func LogStatusBitmaskWithFormat(format string, bitmask *uint32, out io.Writer, next http.Handler) http.Handler {
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
		bm := atomic.LoadUint32(bitmask)

		if bit == 0 || bm&bit > 0 {
			lrw.end = time.Now()
			out.Write([]byte(form(format, lrw, r)))
		}
	})
}

func form(format string, lrw *loggedResponseWriter, r *http.Request) string {
	var buf, token bytes.Buffer
	var capture bool
	var escaped bool
	for _, rune := range format {
		if escaped {
			buf.WriteRune(rune)
			escaped = false
			continue
		}
		if rune == '\\' {
			escaped = true
			continue
		}
		switch capture {
		case false:
			if rune == '{' {
				capture = true
			} else {
				buf.WriteRune(rune)
			}
		case true:
			if rune == '}' {
				switch tok := token.String(); tok {
				case "begin":
					buf.WriteString(lrw.begin.UTC().Format(apacheTimeFormat))
				case "begin-epoch":
					buf.WriteString(strconv.FormatInt(lrw.begin.UTC().Unix(), 10))
				case "begin-iso8601":
					buf.WriteString(lrw.begin.UTC().Format(time.RFC3339))
				case "bytes":
					buf.WriteString(fmt.Sprintf("%d", lrw.responseBytes))
				case "client":
					buf.WriteString(r.RemoteAddr)
				case "client-ip":
					value := r.RemoteAddr // client ip:port
					if colon := strings.LastIndex(value, ":"); colon != -1 {
						value = value[:colon]
					}
					buf.WriteString(value)
				case "client-port":
					value := r.RemoteAddr // client ip:port
					if colon := strings.LastIndex(value, ":"); colon != -1 {
						value = value[colon+1:]
					}
					buf.WriteString(value)
				case "duration":
					// 6 decimal places: microsecond precision
					buf.WriteString(strconv.FormatFloat(lrw.end.Sub(lrw.begin).Seconds(), 'f', 6, 64))
				case "end":
					buf.WriteString(lrw.end.UTC().Format(apacheTimeFormat))
				case "end-epoch":
					buf.WriteString(strconv.FormatInt(lrw.end.UTC().Unix(), 10))
				case "end-iso8601":
					buf.WriteString(lrw.end.UTC().Format(time.RFC3339))
				case "method":
					buf.WriteString(r.Method)
				case "proto":
					buf.WriteString(r.Proto)
				case "status":
					buf.WriteString(strconv.FormatInt(int64(lrw.status), 10))
				case "uri":
					buf.WriteString(r.RequestURI)
				default:
					if strings.HasPrefix(tok, "http-") {
						// emit value of specified HTTP request header
						buf.WriteString(r.Header.Get(tok[5:]))
					} else {
						buf.WriteRune('{')
						buf.WriteString(tok)
						buf.WriteRune(rune)
					}
				}
				token.Reset()
				capture = false
			} else {
				token.WriteRune(rune)
			}
		}
	}
	if capture {
		buf.WriteRune('{')
		buf.Write(token.Bytes())
	}
	buf.WriteRune('\n')
	return buf.String()
}
