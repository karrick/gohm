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
		bm := atomic.LoadUint32(bitmask)

		if bit == 0 || bm&bit > 0 {
			lrw.end = time.Now()
			buf := new(bytes.Buffer)
			for _, emitter := range emitters {
				if _, err := emitter(lrw, r, buf); err != nil {
					panic(err) // TODO
				}
			}
			if _, err := out.Write(buf.Bytes()); err != nil {
				panic(err) // TODO
			}
		}
	})
}

func compileFormat(format string) []func(*loggedResponseWriter, *http.Request, io.Writer) (int, error) {
	// build slice of emitter functions, each will emit the requested information into the output
	var emitters []func(*loggedResponseWriter, *http.Request, io.Writer) (int, error)

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
			// stop capturing buf, and start capturing token
			if capturingToken {
				// is this an error? it was not before
			}
			emitters = append(emitters, makeStringEmitter(buf.String()))
			buf.Reset()
			capturingToken = true
		} else if rune == '}' {
			// stop capturing token, and start capturing buffer
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

func makeStringEmitter(value string) func(*loggedResponseWriter, *http.Request, io.Writer) (int, error) {
	return func(_ *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
		return iow.Write([]byte(value))
	}
}

func beginEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(lrw.begin.UTC().Format(apacheTimeFormat)))
}

func beginEpochEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(strconv.FormatInt(lrw.begin.UTC().Unix(), 10)))
}

func beginISO8601Emitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(lrw.begin.UTC().Format(time.RFC3339)))
}

func bytesEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(fmt.Sprintf("%d", lrw.responseBytes)))
}

func clientEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(r.RemoteAddr))
}

func clientIPEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[:colon]
	}
	return iow.Write([]byte(value))
}

func clientPortEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[colon+1:]
	}
	return iow.Write([]byte(value))
}

func durationEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	// 6 decimal places: microsecond precision
	return iow.Write([]byte(strconv.FormatFloat(lrw.end.Sub(lrw.begin).Seconds(), 'f', 6, 64)))
}

func endEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(lrw.end.UTC().Format(apacheTimeFormat)))
}

func endEpochEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(strconv.FormatInt(lrw.end.UTC().Unix(), 10)))
}

func endISO8601Emitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(lrw.end.UTC().Format(time.RFC3339)))
}

func methodEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(r.Method))
}

func protoEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(r.Proto))
}

func statusEmitter(lrw *loggedResponseWriter, _ *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(strconv.FormatInt(int64(lrw.status), 10)))
}

func uriEmitter(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
	return iow.Write([]byte(r.RequestURI))
}

func makeHeaderEmitter(headerName string) func(*loggedResponseWriter, *http.Request, io.Writer) (int, error) {
	return func(_ *loggedResponseWriter, r *http.Request, iow io.Writer) (int, error) {
		return iow.Write([]byte(r.Header.Get(headerName)))
	}
}
