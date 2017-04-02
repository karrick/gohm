package gohm

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultLogFormat is the default log line format used by this library.
const DefaultLogFormat = "{client-ip} [{begin-iso8601}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"

// ApacheCommonLogFormat (CLF) is the default log line format for Apache Web Server.  It is included
// here for users of this library that would like to easily specify log lines out to be emitted
// using the Apache Common Log Format (CLR).
//
//	"%h %l %u %t \"%r\" %>s %b"
//	"{remote-hostname} {remote-logname} {remote-user} {begin-time} \"{first-line-of-request}\" {status} {bytes}"
//	"{remote-ip} - - {begin-time} \"{first-line-of-request}\" {status} {bytes}"
const ApacheCommonLogFormat = "{client-ip} - - [{begin}] \"{method} {uri} {proto}\" {status} {bytes}"

const apacheTimeFormat = "02/Jan/2006:15:04:05 -0700"

const (
	LogStatus1xx    uint32 = 1                  // LogStatus1xx used to log HTTP requests which have a 1xx response
	LogStatus2xx    uint32 = 2                  // LogStatus2xx used to log HTTP requests which have a 2xx response
	LogStatus3xx    uint32 = 4                  // LogStatus3xx used to log HTTP requests which have a 3xx response
	LogStatus4xx    uint32 = 8                  // LogStatus4xx used to log HTTP requests which have a 4xx response
	LogStatus5xx    uint32 = 16                 // LogStatus5xx used to log HTTP requests which have a 5xx response
	LogStatusAll    uint32 = 1 | 2 | 4 | 8 | 16 // LogStatusAll used to log all HTTP requests
	LogStatusErrors uint32 = 8 | 16             // LogStatusAll used to log HTTP requests which have 4xx or 5xx response
)

type loggedResponseWriter struct {
	http.ResponseWriter
	responseBytes int64
	status        int
	begin, end    time.Time
	closeNotifyCh <-chan bool
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

func (r *loggedResponseWriter) CloseNotify() <-chan bool {
	if r.closeNotifyCh != nil {
		return r.closeNotifyCh
	}
	if notifier, ok := r.ResponseWriter.(http.CloseNotifier); ok {
		r.closeNotifyCh = notifier.CloseNotify()
	} else {
		// Return a channel that nothing will ever emit to, and will eventually be garbage
		// collected.
		//
		// NOTE: I am not absolutely certain about the client side-effects of essentially
		// broadcasting this as a CloseNotifier when it returns a dummy channel.  Well
		// behaved http.Handler functions will attempt to take advantage of the feature, but
		// it will sadly not work.  This can happen when a server program inserts a
		// http.Handler into the pipeline for a call that inserts its own
		// http.ResponseHandler that does not have the CloseNotify method.
		r.closeNotifyCh = make(<-chan bool)
	}
	return r.closeNotifyCh
}

// LogAll returns a new http.Handler that logs HTTP requests and responses using the
// gohm.DefaultLogFormat to the specified io.Writer.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.LogAll(os.Stderr, someHandler))
func LogAll(out io.Writer, next http.Handler) http.Handler {
	logBitmask := LogStatusAll
	return LogStatusBitmaskWithFormat(DefaultLogFormat, &logBitmask, out, next)
}

// LogAllWithFormat returns a new http.Handler that logs HTTP requests and responses using the
// specified log format string to the specified io.Writer.
//
//	mux := http.NewServeMux()
//	format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//	mux.Handle("/example/path", gohm.LogAllWithFormat(format, os.Stderr, someHandler))
func LogAllWithFormat(format string, out io.Writer, next http.Handler) http.Handler {
	logBitmask := LogStatusAll
	return LogStatusBitmaskWithFormat(format, &logBitmask, out, next)
}

// LogErrors returns a new http.Handler that logs HTTP requests that result in response errors, or
// more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output lines
// using the gohm.DefaultLogFormat to the specified io.Writer.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.LogErrors(os.Stderr, someHandler))
func LogErrors(out io.Writer, next http.Handler) http.Handler {
	logBitmask := LogStatusErrors
	return LogStatusBitmaskWithFormat(DefaultLogFormat, &logBitmask, out, next)
}

// LogErrorsWithFormat returns a new http.Handler that logs HTTP requests that result in response
// errors, or more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will
// output lines using the specified log format string to the specified io.Writer.
//
//	mux := http.NewServeMux()
//	format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//	mux.Handle("/example/path", gohm.LogErrorsWithFormat(format, os.Stderr, someHandler))
func LogErrorsWithFormat(format string, out io.Writer, next http.Handler) http.Handler {
	logBitmask := LogStatusErrors
	return LogStatusBitmaskWithFormat(format, &logBitmask, out, next)
}

// LogStatusBitmask returns a new http.Handler that logs HTTP requests that have a status code that
// matches any of the status codes in the specified bitmask.  The handler will output lines using
// the gohm.DefaultLogFormat to the specified io.Writer.
//
// The bitmask parameter is used to specify which HTTP requests ought to be logged based on the HTTP
// status code returned by the next http.Handler.
//
//	mux := http.NewServeMux()
//	logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
//	mux.Handle("/example/path", gohm.LogStatusBitmask(&logBitmask, os.Stderr, someHandler))
func LogStatusBitmask(bitmask *uint32, out io.Writer, next http.Handler) http.Handler {
	return LogStatusBitmaskWithFormat(DefaultLogFormat, bitmask, out, next)
}

// LogStatusBitmaskWithFormat returns a new http.Handler that logs HTTP requests that have a status
// code that matches any of the status codes in the specified bitmask.  The handler will output
// lines in the specified log format string to the specified io.Writer.
//
// The following format directives are supported:
//
//	begin:           time request received (apache log time format)
//	begin-epoch:     time request received (epoch)
//	begin-iso8601:   time request received (ISO-8601 time format)
//	bytes:           response size
//	client:          client-ip:client-port
//	client-ip:       client IP address
//	client-port:     client port
//	duration:        duration of request from begin to end, (seconds with millisecond precision)
//	end:             time request completed (apache log time format)
//	end-epoch:       time request completed (epoch)
//	end-iso8601:     time request completed (ISO-8601 time format)
//	method:          request method, e.g., GET or POST
//	proto:           request protocol, e.g., HTTP/1.1
//	status:          response status code
//	uri:             request URI
//
// In addition, values from HTTP request headers can also be included in the log by prefixing the
// HTTP header name with http-.  In the below example, each log line will begin with the value of
// the HTTP request header CLIENT-IP followed by the value of the HTTP request header USER:
//
//	format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//
// The bitmask parameter is used to specify which HTTP requests ought to be logged based on the HTTP
// status code returned by the next http.Handler.
//
//	mux := http.NewServeMux()
//	format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
//	logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
//	mux.Handle("/example/path", gohm.LogStatusBitmaskWithFormat(format, &logBitmask, os.Stderr, someHandler))
func LogStatusBitmaskWithFormat(format string, bitmask *uint32, out io.Writer, next http.Handler) http.Handler {
	emitters := compileFormat(format)

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
			buf := new(bytes.Buffer)
			for _, emitter := range emitters {
				emitter(lrw, r, buf)
			}
			if _, err := out.Write(buf.Bytes()); err != nil {
				// if we cannot write to out for some reason, write it to stderr
				_, _ = buf.WriteTo(os.Stderr)
			}
		}
	})
}

func compileFormat(format string) []func(*loggedResponseWriter, *http.Request, *bytes.Buffer) {
	// build slice of emitter functions, each will emit the requested information into the output
	var emitters []func(*loggedResponseWriter, *http.Request, *bytes.Buffer)

	// state machine alternating between two states: either capturing runes for the next
	// constant buffer, or capturing runes for the next token
	var buf, token bytes.Buffer
	var capturingToken bool

	var nextRuneEscaped bool // true when next rune has been escaped

	for _, rune := range format {
		if nextRuneEscaped {
			// when this rune has been escaped, then just write it out to whichever
			// buffer we're collecting to right now
			if capturingToken {
				token.WriteRune(rune)
			} else {
				buf.WriteRune(rune)
			}
			nextRuneEscaped = false
			continue
		}
		if rune == '\\' {
			// format specifies that next rune ought to be escaped
			nextRuneEscaped = true
			continue
		}
		if rune == '{' {
			// stop capturing buf, and begin capturing token
			if capturingToken {
				// is this an error? it was not before
			}
			emitters = append(emitters, makeStringEmitter(buf.String()))
			buf.Reset()
			capturingToken = true
		} else if rune == '}' {
			// stop capturing token, and begin capturing buffer
			if !capturingToken {
				// is this an error?
			}
			switch tok := token.String(); tok {
			case "begin":
				emitters = append(emitters, beginEmitter)
			case "begin-epoch":
				emitters = append(emitters, beginEpochEmitter)
			case "begin-iso8601":
				emitters = append(emitters, beginISO8601Emitter)
			case "bytes":
				emitters = append(emitters, bytesEmitter)
			case "client":
				emitters = append(emitters, clientEmitter)
			case "client-ip":
				emitters = append(emitters, clientIPEmitter)
			case "client-port":
				emitters = append(emitters, clientPortEmitter)
			case "duration":
				emitters = append(emitters, durationEmitter)
			case "end":
				emitters = append(emitters, endEmitter)
			case "end-epoch":
				emitters = append(emitters, endEpochEmitter)
			case "end-iso8601":
				emitters = append(emitters, endISO8601Emitter)
			case "method":
				emitters = append(emitters, methodEmitter)
			case "proto":
				emitters = append(emitters, protoEmitter)
			case "status":
				emitters = append(emitters, statusEmitter)
			case "uri":
				emitters = append(emitters, uriEmitter)
			default:
				if strings.HasPrefix(tok, "http-") {
					// emit value of specified HTTP request header
					emitters = append(emitters, makeHeaderEmitter(tok[5:]))
				} else {
					// unknown token, so just append to buf
					buf.WriteRune('{')
					buf.WriteString(tok)
					buf.WriteRune(rune)
				}
			}
			token.Reset()
			capturingToken = false
		} else {
			// emit to either token or buffer
			if capturingToken {
				token.WriteRune(rune)
			} else {
				buf.WriteRune(rune)
			}
		}
	}
	if capturingToken {
		buf.WriteRune('{')
		buf.Write(token.Bytes())
	}
	buf.WriteRune('\n')
	emitters = append(emitters, makeStringEmitter(buf.String()))

	return emitters
}

func makeStringEmitter(value string) func(*loggedResponseWriter, *http.Request, *bytes.Buffer) {
	return func(_ *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
		bb.WriteString(value)
	}
}

func beginEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.begin.UTC().Format(apacheTimeFormat))
}

func beginEpochEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(lrw.begin.UTC().Unix(), 10))
}

func beginISO8601Emitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.begin.UTC().Format(time.RFC3339))
}

func bytesEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(lrw.responseBytes, 10))
}

func clientEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.RemoteAddr)
}

func clientIPEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[:colon]
	}
	bb.WriteString(value)
}

func clientPortEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[colon+1:]
	}
	bb.WriteString(value)
}

func durationEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	// 6 decimal places: microsecond precision
	bb.WriteString(strconv.FormatFloat(lrw.end.Sub(lrw.begin).Seconds(), 'f', 6, 64))
}

func endEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.end.UTC().Format(apacheTimeFormat))
}

func endEpochEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(lrw.end.UTC().Unix(), 10))
}

func endISO8601Emitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.end.UTC().Format(time.RFC3339))
}

func methodEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.Method)
}

func protoEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.Proto)
}

func statusEmitter(lrw *loggedResponseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(int64(lrw.status), 10))
}

func uriEmitter(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.RequestURI)
}

func makeHeaderEmitter(headerName string) func(*loggedResponseWriter, *http.Request, *bytes.Buffer) {
	return func(_ *loggedResponseWriter, r *http.Request, bb *bytes.Buffer) {
		bb.WriteString(r.Header.Get(headerName))
	}
}
