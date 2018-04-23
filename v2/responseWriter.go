package gohm

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/karrick/gorill"
)

// responseWriter must behave exactly like http.ResponseWriter, yet store up
// response until query complete and flush invoked.
type responseWriter struct {
	http.ResponseWriter
	header        http.Header
	body          bytes.Buffer
	size          int64
	status        int
	statusWritten bool
	errorMessage  string
	begin, end    time.Time
}

func (rw *responseWriter) Header() http.Header {
	m := rw.header
	if m == nil {
		m = make(http.Header)
		rw.header = m
	}
	return m
}

func (rw *responseWriter) Write(blob []byte) (int, error) {
	return rw.body.Write(blob)
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.statusWritten = true
}

// update responseWriter then enqueue status and message to be send to client
func (rw *responseWriter) error(message string, status int) {
	rw.errorMessage = message
	rw.status = status
	Error(rw, rw.errorMessage, rw.status)
}

func (rw *responseWriter) flush() error {
	// write header
	header := rw.ResponseWriter.Header()
	for key, values := range rw.header {
		for _, value := range values {
			header.Add(key, value)
		}
	}

	// write status
	if !rw.statusWritten {
		rw.status = http.StatusOK
	}
	rw.ResponseWriter.WriteHeader(rw.status)

	// write response
	var err error

	// NOTE: Apache Common Log Format size excludes HTTP headers
	rw.size, err = rw.body.WriteTo(rw.ResponseWriter)
	return err
}

// New returns a new http.Handler that calls the specified next http.Handler,
// and performs the requested operations before and after the downstream handler
// as specified by the gohm.Config structure passed to it.
//
// It receives a gohm.Config struct rather than a pointer to one, so users less
// likely to consider modification after creating the http.Handler.
//
//  // Used to control how long it takes to serve a static file.
//	const staticTimeout = time.Second
//
//	var (
//		// Will store statistics counters for status codes 1xx, 2xx, 3xx, 4xx,
//		// 5xx, as well as a counter for all responses
//		counters gohm.Counters
//
//		// Used to dynamically control log level of HTTP logging. After handler
//      // created, this must be accessed using the sync/atomic package.
//		logBitmask = gohm.LogStatusErrors
//
//		// Determines HTTP log format
//		logFormat = "{http-CLIENT-IP} {client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {message}"
//	)
//
//	func main() {
//
//		h := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
//
//		h = gohm.WithGzip(h) // gzip response if client accepts gzip encoding
//
//		// gohm was designed to wrap other http.Handler functions.
//		h = gohm.New(h, gohm.Config{
//			Counters:   &counters,   // pointer given so counters can be collected and optionally reset
//			LogBitmask: &logBitmask, // pointer given so bitmask can be updated using sync/atomic
//			LogFormat:  logFormat,
//			LogWriter:  os.Stderr,
//			Timeout:    staticTimeout,
//		})
//
//		http.Handle("/static/", h)
//		log.Fatal(http.ListenAndServe(":8080", nil))
//	}
func New(next http.Handler, config Config) http.Handler {
	var emitters []func(*responseWriter, *http.Request, *[]byte)

	if config.LogWriter != nil {
		if config.LogBitmask == nil {
			// Set a default bitmask to log all requests
			logBitmask := LogStatusAll
			config.LogBitmask = &logBitmask
		}
		if config.LogFormat == "" {
			// Set a default log line format
			config.LogFormat = DefaultLogFormat
		}
		emitters = compileFormat(config.LogFormat)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var er *gorill.EscrowReader

		if config.EscrowReader {
			var bb *bytes.Buffer
			if config.BufPool != nil {
				// Obtain a bytes.Buffer from the buffer pool, but use defer to
				// return the used buffer to the pool because we cannot control
				// whether the specified next http.Handler or callback function
				// will panic.
				bb = config.BufPool.Get()
				defer config.BufPool.Put(bb)
			}

			// Only pre-allocate buffer when the Content-Length parses. It would
			// be really nice to provide some signaling mechanism to the next
			// handler when the Content-Length fails to parse, but I suppose if
			// it cares, it will also check the same thing.
			if contentLengthString := r.Header.Get("Content-Length"); contentLengthString != "" {
				if contentLength, err := strconv.Atoi(contentLengthString); err == nil {
					if bb != nil {
						// Ensure existing buffer is large enough to read
						// Content-Length bytes.
						bb.Grow(contentLength)
					} else {
						// Pre-allocate a buffer large enough to read
						// Content-Length bytes.
						bb = bytes.NewBuffer(make([]byte, 0, contentLength))
					}
				}
			}

			// Update the original request's Body to point to a newly created
			// structure, from which the next handler may read the request body,
			// and from which the callback may also read the request body if
			// required.
			er = gorill.NewEscrowReader(r.Body, bb)
			r.Body = er
		}

		// Create a responseWriter to pass to next.ServeHTTP and collect
		// downstream handler's response to query.  It will eventually be used
		// to flush to the client, assuming neither the handler panics, nor the
		// client connection is detected to be closed.
		rw := &responseWriter{ResponseWriter: w}

		var ctx context.Context
		var cancel func()

		// Create a couple of channels to detect one of 3 ways to exit this
		// handler.
		serverCompleted := make(chan struct{})
		serverPanicked := make(chan string, 1)

		if config.Timeout > 0 {
			// Adding a timeout to a request context spins off a goroutine that
			// will invoke the specified cancel function for us after the
			// timeout has elapsed.  Invoking the cancel function causes the
			// context's Done channel to close.  Detecting timeout is done by
			// waiting for context.Done() to close.
			ctx, cancel = context.WithTimeout(r.Context(), config.Timeout)
		} else {
			// When no timeout given, we still need a mechanism to track context
			// cancellation so this handler can detect when client has closed
			// its connection.
			ctx, cancel = context.WithCancel(r.Context())
		}
		r = r.WithContext(ctx)
		defer cancel()

		rw.begin = time.Now()

		// We must invoke downstream handler in separate goroutine in order to
		// ensure this handler only responds to one of the three events below,
		// whichever event takes place first.
		go serveWithPanicProtection(rw, r, next, serverCompleted, serverPanicked)

		// Wait for the first of either of 3 events:
		//   * serveComplete: the next.ServeHTTP method completed normally
		//     (possibly even with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and
		//     panicked instead with a text message.
		//   * context is done: triggered when timeout or client disconnect.
		select {

		case <-serverCompleted:
			// break

		case text := <-serverPanicked:
			if config.AllowPanics {
				panic(text) // do not need to tell downstream to cancel, because it already panicked.
			}
			rw.error(text, http.StatusInternalServerError)

		case <-ctx.Done():
			// we'll create a new rw that downstream handler doesn't have access
			// to so it cannot mutate it.
			rw = &responseWriter{ResponseWriter: w, begin: rw.begin}

			// the context was canceled; where ctx.Err() will say why 503 (this
			// is what http.TimeoutHandler returns)
			rw.error(ctx.Err().Error(), http.StatusServiceUnavailable)

		}

		if err := rw.flush(); err != nil {
			// cannot write responseWriter's contents to http.ResponseWriter
			rw.errorMessage = err.Error()
			rw.status = http.StatusInternalServerError
			// no use emitting error message to client when cannot send original
			// payload back
		}

		rw.end = time.Now()
		statusClass := rw.status / 100

		// Update status counters
		if config.Counters != nil {
			atomic.AddUint64(&config.Counters.counters[0], 1)           // all
			atomic.AddUint64(&config.Counters.counters[statusClass], 1) // 1xx, 2xx, 3xx, 4xx, 5xx
		}

		// Invoke callback if provided, prior to logging request.
		var stats *Statistics
		if config.Callback != nil {
			stats = &Statistics{
				RequestBegin:   rw.begin,
				ResponseStatus: rw.status,
				ResponseEnd:    rw.end,
			}
			if er != nil {
				stats.RequestBody = er.Bytes()
			}
			config.Callback(stats)
		}

		// Update log
		if config.LogWriter != nil {
			if (stats != nil && stats.emitLog) || (atomic.LoadUint32(config.LogBitmask))&(1<<uint32(statusClass-1)) > 0 {
				buf := make([]byte, 0, 128)
				for _, emitter := range emitters {
					emitter(rw, r, &buf)
				}
				_, _ = config.LogWriter.Write(buf)
			}
		}
	})
}