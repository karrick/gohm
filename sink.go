package gohm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	CloseNotifier         bool
	ConvertPanicsToErrors bool
	Counters              *Counters
	LogBitmask            uint32
	LogFormat             string
	LogWriter             io.Writer
	Timeout               time.Duration
}

func Sink(next http.Handler, config *Config) http.Handler {
	var emitters []func(*responseWriter, *http.Request, *bytes.Buffer)
	if config.LogWriter != nil {
		emitters = compileFormatSink(config.LogFormat)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a responseWriter to pass to next.ServeHTTP and collect downstream
		// handler's response to query.  It will eventually be used to flush to the client,
		// assuming neither the handler panics, nor the client connection is detected to be
		// closed.
		rw := &responseWriter{ResponseWriter: w}

		ctx := r.Context()
		var cancel func()

		// Create a couple of channels to detect one of 3 ways to exit this handler.
		clientDisconnected := make(chan struct{})
		serverCompleted := make(chan struct{})
		serverPanicked := make(chan string, 1)
		timedOut := make(chan struct{})

		if config.CloseNotifier {
			// Not all http.ResponseHandlers implement http.CloseNotifier.  If the
			// http.ResponseHandler we were given does, then we can use it to detect when the
			// client has closed its connection socket.  If the http.ResponseWriter does not
			// implement http.CloseNotifier, there same overhead applies, but the downstream
			// handlers will still work correctly, however, this handler simply will not detect
			// when the client has closed the connection.
			if notifier, ok := w.(http.CloseNotifier); ok {
				receivingBlocksUntilRemoteClosed := notifier.CloseNotify()
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				r = r.WithContext(ctx)

				// Watchdog goroutine sits and waits for the client to possibly close the
				// connection, and trigger required actions if it does.
				go func() {
					<-receivingBlocksUntilRemoteClosed
					// When here, the remote has closed connection.

					// Tell downstream it may stop trying to serve the request.  Many
					// handlers still ignore context cancellations, but we do what we
					// can.
					cancel()

					// Terminate this handler, and if logger attached upstream, let's
					// throw in a descriptive server log message
					close(clientDisconnected)
				}()
			}
		}

		if config.Timeout > 0 {
			// Watchdog goroutine sits and waits for the timeout to expire and trigger required
			// actions if it does.
			go func() {
				time.Sleep(config.Timeout)
				close(timedOut)
			}()

			// While not all handlers use context and would respect timeout, it's likely that
			// more and more will over time as context becomes more popular.  Even though this
			// handler will handle the timeout, we modify the context so any context-aware
			// handlers downstream will get the signal when the timeout has elapsed.
			ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
			defer cancel()
			r = r.WithContext(ctx)
		}

		if config.LogWriter != nil {
			rw.begin = time.Now()
		}

		// We must invoke downstream handler in separate goroutine in order to ensure this
		// handler only responds to one of the three events below, whichever event takes
		// place first.
		go serveWithPanicProtection(rw, r, next, serverCompleted, serverPanicked)

		// Wait for the first of either of 3 events:
		//   * serveComplete: the next.ServeHTTP method completed normally (possibly even
		//     with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and panicked
		//     instead with a text message.
		//   * context is done: triggered when timeout has been exceeded.
		select {

		case <-serverCompleted:
			if err := rw.flush(); err != nil {
				Error(w, fmt.Sprintf("cannot flush response writer: %s", err), http.StatusInternalServerError)
			}

		case text := <-serverPanicked:
			// Error(w, text, http.StatusInternalServerError)

			// NOTE: While this could simply emit the error message here, right now it
			// re-panics from this goroutine, effectively capturing and replaying the
			// panic from the downstream handler that took place in a different
			// goroutine.
			panic(text) // do not need to tell downstream to cancel, because it already panicked.

		case <-timedOut:
			// timeout watchdog routine triggered
			Error(w, "took too long to process request", http.StatusServiceUnavailable) // 503 (this is what http.TimeoutHandler returns)

		case <-ctx.Done():
			// the context was canceled; where ctx.Err() will say why
			Error(w, ctx.Err().Error(), http.StatusServiceUnavailable) // 503 (this is what http.TimeoutHandler returns)

		case <-clientDisconnected:
			Error(w, "cannot serve to disconnected client", http.StatusRequestTimeout) // 408 (or should we use another?)

		}

		statusClass := rw.status / 100

		if config.Counters != nil {
			atomic.AddUint64(&config.Counters.counterAll, 1)
			switch statusClass {
			case 1:
				atomic.AddUint64(&config.Counters.counter1xx, 1)
			case 2:
				atomic.AddUint64(&config.Counters.counter2xx, 1)
			case 3:
				atomic.AddUint64(&config.Counters.counter3xx, 1)
			case 4:
				atomic.AddUint64(&config.Counters.counter4xx, 1)
			case 5:
				atomic.AddUint64(&config.Counters.counter5xx, 1)
			}
		}

		// LOG
		if config.LogWriter != nil {
			var bit uint32
			switch statusClass {
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

			if bit == 0 || (atomic.LoadUint32(&config.LogBitmask))&bit > 0 {
				rw.end = time.Now()

				buf := new(bytes.Buffer)
				for _, emitter := range emitters {
					emitter(rw, r, buf)
				}
				if _, err := buf.WriteTo(config.LogWriter); err != nil {
					// if we cannot write to out for some reason, write it to stderr
					_, _ = buf.WriteTo(os.Stderr)
				}
			}

		}

	})
}

func compileFormatSink(format string) []func(*responseWriter, *http.Request, *bytes.Buffer) {
	// build slice of emitter functions, each will emit the requested information into the output
	var emitters []func(*responseWriter, *http.Request, *bytes.Buffer)

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
			emitters = append(emitters, makeStringEmitterSink(buf.String()))
			buf.Reset()
			capturingToken = true
		} else if rune == '}' {
			// stop capturing token, and begin capturing buffer
			if !capturingToken {
				// is this an error?
			}
			switch tok := token.String(); tok {
			case "begin":
				emitters = append(emitters, beginEmitterSink)
			case "begin-epoch":
				emitters = append(emitters, beginEpochEmitterSink)
			case "begin-iso8601":
				emitters = append(emitters, beginISO8601EmitterSink)
			case "bytes":
				emitters = append(emitters, bytesEmitterSink)
			case "client":
				emitters = append(emitters, clientEmitterSink)
			case "client-ip":
				emitters = append(emitters, clientIPEmitterSink)
			case "client-port":
				emitters = append(emitters, clientPortEmitterSink)
			case "duration":
				emitters = append(emitters, durationEmitterSink)
			case "end":
				emitters = append(emitters, endEmitterSink)
			case "end-epoch":
				emitters = append(emitters, endEpochEmitterSink)
			case "end-iso8601":
				emitters = append(emitters, endISO8601EmitterSink)
			case "method":
				emitters = append(emitters, methodEmitterSink)
			case "proto":
				emitters = append(emitters, protoEmitterSink)
			case "status":
				emitters = append(emitters, statusEmitterSink)
			case "uri":
				emitters = append(emitters, uriEmitterSink)
			default:
				if strings.HasPrefix(tok, "http-") {
					// emit value of specified HTTP request header
					emitters = append(emitters, makeHeaderEmitterSink(tok[5:]))
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
	emitters = append(emitters, makeStringEmitterSink(buf.String()))

	return emitters
}

func makeStringEmitterSink(value string) func(*responseWriter, *http.Request, *bytes.Buffer) {
	return func(_ *responseWriter, _ *http.Request, bb *bytes.Buffer) {
		bb.WriteString(value)
	}
}

func beginEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.begin.UTC().Format(apacheTimeFormat))
}

func beginEpochEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(lrw.begin.UTC().Unix(), 10))
}

func beginISO8601EmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.begin.UTC().Format(time.RFC3339))
}

func bytesEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(int64(lrw.body.Len()), 10))
}

func clientEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.RemoteAddr)
}

func clientIPEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[:colon]
	}
	bb.WriteString(value)
}

func clientPortEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	value := r.RemoteAddr // ip:port
	if colon := strings.LastIndex(value, ":"); colon != -1 {
		value = value[colon+1:]
	}
	bb.WriteString(value)
}

func durationEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	// 6 decimal places: microsecond precision
	bb.WriteString(strconv.FormatFloat(lrw.end.Sub(lrw.begin).Seconds(), 'f', 6, 64))
}

func endEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.end.UTC().Format(apacheTimeFormat))
}

func endEpochEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(lrw.end.UTC().Unix(), 10))
}

func endISO8601EmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(lrw.end.UTC().Format(time.RFC3339))
}

func methodEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.Method)
}

func protoEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.Proto)
}

func statusEmitterSink(lrw *responseWriter, _ *http.Request, bb *bytes.Buffer) {
	bb.WriteString(strconv.FormatInt(int64(lrw.status), 10))
}

func uriEmitterSink(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
	bb.WriteString(r.RequestURI)
}

func makeHeaderEmitterSink(headerName string) func(*responseWriter, *http.Request, *bytes.Buffer) {
	return func(_ *responseWriter, r *http.Request, bb *bytes.Buffer) {
		bb.WriteString(r.Header.Get(headerName))
	}
}
