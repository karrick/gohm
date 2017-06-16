package gohm

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ApacheCommonLogFormat (CLF) is the default log line format for Apache Web
// Server.  It is included here for users of this library that would like to
// easily specify log lines out to be emitted using the Apache Common Log Format
// (CLR), by setting `LogFormat` to `gohm.ApackeCommonLogFormat`.
const ApacheCommonLogFormat = "{client-ip} - - [{begin}] \"{method} {uri} {proto}\" {status} {bytes}"

const apacheTimeFormat = "02/Jan/2006:15:04:05 -0700"

// NOTE: Apache Common Log Format size excludes HTTP headers
// "%h %l %u %t \"%r\" %>s %b"
// "{remote-hostname} {remote-logname} {remote-user} {begin-time} \"{first-line-of-request}\" {status} {bytes}"
// "{remote-ip} - - {begin-time} \"{first-line-of-request}\" {status} {bytes}"

// DefaultLogFormat is the default log line format used by this library.
const DefaultLogFormat = "{client-ip} [{begin-iso8601}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {error}"

// LogStatus1xx used to log HTTP requests which have a 1xx response
const LogStatus1xx uint32 = 1

// LogStatus2xx used to log HTTP requests which have a 2xx response
const LogStatus2xx uint32 = 2

// LogStatus3xx used to log HTTP requests which have a 3xx response
const LogStatus3xx uint32 = 4

// LogStatus4xx used to log HTTP requests which have a 4xx response
const LogStatus4xx uint32 = 8

// LogStatus5xx used to log HTTP requests which have a 5xx response
const LogStatus5xx uint32 = 16

// LogStatusAll used to log all HTTP requests
const LogStatusAll uint32 = 1 | 2 | 4 | 8 | 16

// LogStatusErrors used to log HTTP requests which have 4xx or 5xx response
const LogStatusErrors uint32 = 8 | 16

// compileFormat converts the format string into a slice of functions to invoke
// when creating a log line.  It's implemented as a state machine that
// alternates between 2 states: consuming runes to create a constant string to
// emit, and consuming runes to create a token that is intended to match one of
// the pre-defined format specifier tokens, or an undefined format specifier
// token that begins with "http-".
func compileFormat(format string) []func(*responseWriter, *http.Request, *[]byte) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(*responseWriter, *http.Request, *[]byte)

	// state machine alternating between two states: either capturing runes for
	// the next constant buffer, or capturing runes for the next token
	var buf, token bytes.Buffer
	var capturingToken bool  // false, because start off capturing buffer runes
	var nextRuneEscaped bool // true when next rune has been escaped

	for _, rune := range format {
		if nextRuneEscaped {
			// when this rune has been escaped, then just write it out to
			// whichever buffer we're collecting to right now
			if capturingToken {
				token.WriteRune(rune)
			} else {
				buf.WriteRune(rune)
			}
			nextRuneEscaped = false
			continue
		}
		if rune == '\\' {
			// Format specifies that next rune ought to be escaped.  Handy when
			// extra curly braces are desired in the log line format.
			nextRuneEscaped = true
			continue
		}
		if rune == '{' {
			// Stop capturing buf, and begin capturing token.
			// NOTE: undefined behavior if open curly brace when previous open
			// curly brace has not yet been closed.
			emitters = append(emitters, makeStringEmitter(buf.String()))
			buf.Reset()
			capturingToken = true
		} else if rune == '}' {
			// Stop capturing token, and begin capturing buffer.
			// NOTE: undefined behavior if close curly brace when not capturing
			// runes for a token.
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
			case "error":
				emitters = append(emitters, errorMessageEmitter)
			case "method":
				emitters = append(emitters, methodEmitter)
			case "proto":
				emitters = append(emitters, protoEmitter)
			case "status":
				emitters = append(emitters, statusEmitter)
			case "status-text":
				emitters = append(emitters, statusTextEmitter)
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

func makeStringEmitter(value string) func(*responseWriter, *http.Request, *[]byte) {
	return func(_ *responseWriter, _ *http.Request, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func beginEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, lrw.begin.UTC().Format(apacheTimeFormat)...)
}

func beginEpochEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(lrw.begin.UTC().Unix(), 10)...)
}

func beginISO8601Emitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, lrw.begin.UTC().Format(time.RFC3339)...)
}

func bytesEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(lrw.size, 10)...)
}

func clientEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.RemoteAddr...)
}

func clientIPEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[:colon]
	}
	*bb = append(*bb, value...)
}

func clientPortEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[colon+1:]
	}
	*bb = append(*bb, value...)
}

func durationEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	// 6 decimal places: microsecond precision
	*bb = append(*bb, strconv.FormatFloat(lrw.end.Sub(lrw.begin).Seconds(), 'f', 6, 64)...)
}

func endEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, lrw.end.UTC().Format(apacheTimeFormat)...)
}

func endEpochEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(lrw.end.UTC().Unix(), 10)...)
}

func endISO8601Emitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, lrw.end.UTC().Format(time.RFC3339)...)
}

func errorMessageEmitter(rw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, rw.errorMessage...)
}

func methodEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.Method...)
}

func protoEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.Proto...)
}

func statusEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(int64(lrw.status), 10)...)
}

func statusTextEmitter(lrw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, http.StatusText(lrw.status)...)
}

func uriEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.RequestURI...)
}

func makeHeaderEmitter(headerName string) func(*responseWriter, *http.Request, *[]byte) {
	return func(_ *responseWriter, r *http.Request, bb *[]byte) {
		value := r.Header.Get(headerName)
		if value == "" {
			value = "-"
		}
		*bb = append(*bb, value...)
	}
}
