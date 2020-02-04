package gohm

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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
func compileFormat(format string) ([]func(*responseWriter, *http.Request, *[]byte), []string) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(*responseWriter, *http.Request, *[]byte)

	hm := make(map[string]struct{})

	// state machine alternating between two states: either capturing runes for
	// the next constant buffer, or capturing runes for the next token
	var buf, token []byte
	var capturingToken bool  // false, because start off capturing buffer runes
	var nextRuneEscaped bool // true when next rune has been escaped

	for _, rune := range format {
		if nextRuneEscaped {
			// when this rune has been escaped, then just write it out to
			// whichever buffer we're collecting to right now
			if capturingToken {
				appendRune(&token, rune)
			} else {
				appendRune(&buf, rune)
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
			emitters = append(emitters, makeStringEmitter(string(buf)))
			buf = buf[:0]
			capturingToken = true
		} else if rune == '}' {
			// Stop capturing token, and begin capturing buffer.
			// NOTE: undefined behavior if close curly brace when not capturing
			// runes for a token.
			switch tok := string(token); tok {
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
					header := tok[5:]
					hm[header] = struct{}{}
					emitters = append(emitters, makeHeaderEmitter(header))
				} else {
					// unknown token: just append to buf, wrapped in curly
					// braces
					buf = append(buf, '{')
					buf = append(buf, tok...)
					buf = append(buf, '}')
				}
			}
			token = token[:0]
			capturingToken = false
		} else {
			// append to either token or buffer
			if capturingToken {
				appendRune(&token, rune)
			} else {
				appendRune(&buf, rune)
			}
		}
	}
	if capturingToken {
		buf = append(buf, '{') // token started with left curly brace, so it needs to precede the token
		buf = append(buf, token...)
	}
	buf = append(buf, '\n') // each log line terminated by newline byte
	emitters = append(emitters, makeStringEmitter(string(buf)))

	var headers []string
	if l := len(hm); l > 0 {
		headers = make([]string, 0, l)
		for header := range hm {
			headers = append(headers, header)
		}
	}
	return emitters, headers
}

func makeStringEmitter(value string) func(*responseWriter, *http.Request, *[]byte) {
	return func(_ *responseWriter, _ *http.Request, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func beginEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, grw.begin.UTC().Format(apacheTimeFormat)...)
}

func beginEpochEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(grw.begin.UTC().Unix(), 10)...)
}

func beginISO8601Emitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, grw.begin.UTC().Format(time.RFC3339)...)
}

func bytesEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(grw.bytesWritten, 10)...)
}

func clientEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.RemoteAddr...)
}

func clientIPEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	value := []byte(r.RemoteAddr) // "ipv4:port", or "[ipv6]:port"
	// strip port
	if colon := bytes.LastIndexByte(value, ':'); colon != -1 {
		value = value[:colon]
	}
	// strip square brackets
	if l := len(value); l > 2 && value[0] == '[' && value[l-1] == ']' {
		value = value[1 : l-1]
	}
	// append remaining bytes
	*bb = append(*bb, value...)
}

func clientPortEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[colon+1:]
	}
	*bb = append(*bb, value...)
}

func durationEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	// 6 decimal places: microsecond precision
	*bb = append(*bb, strconv.FormatFloat(grw.end.Sub(grw.begin).Seconds(), 'f', 6, 64)...)
}

func endEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, grw.end.UTC().Format(apacheTimeFormat)...)
}

func endEpochEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(grw.end.UTC().Unix(), 10)...)
}

func endISO8601Emitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, grw.end.UTC().Format(time.RFC3339)...)
}

func errorMessageEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, grw.responseError...)
}

func methodEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.Method...)
}

func protoEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.Proto...)
}

func statusEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(int64(grw.responseStatus), 10)...)
}

func statusTextEmitter(grw *responseWriter, _ *http.Request, bb *[]byte) {
	*bb = append(*bb, http.StatusText(grw.responseStatus)...)
}

func uriEmitter(_ *responseWriter, r *http.Request, bb *[]byte) {
	*bb = append(*bb, r.RequestURI...)
}

func makeHeaderEmitter(headerName string) func(*responseWriter, *http.Request, *[]byte) {
	return func(grw *responseWriter, _ *http.Request, bb *[]byte) {
		*bb = append(*bb, grw.requestHeaders[headerName]...)
	}
}

func appendRune(buf *[]byte, r rune) {
	if r < utf8.RuneSelf {
		*buf = append(*buf, byte(r))
		return
	}
	olen := len(*buf)
	*buf = append(*buf, 0, 0, 0, 0)              // grow buf large enough to accommodate largest possible UTF8 sequence
	n := utf8.EncodeRune((*buf)[olen:olen+4], r) // encode rune into newly allocated buf space
	*buf = (*buf)[:olen+n]                       // trim buf to actual size used by rune addition
}
