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
	var loggedHeaders []string

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
		emitters, loggedHeaders = compileFormat(config.LogFormat)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var er *gorill.EscrowReader

		if config.EscrowReader {
			var erb *bytes.Buffer // escrow read buffer
			if config.BufPool != nil {
				// Obtain a bytes.Buffer from the buffer pool, but use defer to
				// return the used buffer to the pool because we cannot control
				// whether the specified next http.Handler or callback function
				// will panic.
				erb = config.BufPool.Get()
				defer config.BufPool.Put(erb)
			}

			// Only pre-allocate buffer when the Content-Length parses. It would
			// be really nice to provide some signaling mechanism to the next
			// handler when the Content-Length fails to parse, but I suppose if
			// it cares, it will also check the same thing.
			if contentLengthString := r.Header.Get("Content-Length"); contentLengthString != "" {
				if contentLength, err := strconv.Atoi(contentLengthString); err == nil {
					if erb != nil {
						// Ensure existing buffer is large enough to read
						// Content-Length bytes.
						erb.Grow(contentLength)
					} else {
						// Pre-allocate a buffer large enough to read
						// Content-Length bytes.
						erb = bytes.NewBuffer(make([]byte, 0, contentLength))
					}
				}
			}

			// Update the original request's Body to point to a newly created
			// structure, from which the next handler may read the request body,
			// and from which the callback may also read the request body if
			// required.
			er = gorill.NewEscrowReader(r.Body, erb)
			r.Body = er
		}

		ctx := r.Context()

		if config.Timeout > 0 {
			// Adding a timeout to a request context spins off a goroutine that
			// will invoke the specified cancel function for us after the
			// timeout has elapsed.  Invoking the cancel function causes the
			// context's Done channel to close.  Detecting timeout is done by
			// waiting for context.Done() to close.
			var cancel func()
			ctx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			r = r.WithContext(ctx)
		}

		var bb *bytes.Buffer
		if config.BufPool != nil {
			bb = config.BufPool.Get()
			defer config.BufPool.Put(bb)
		} else {
			bb = new(bytes.Buffer)
		}

		// Create a responseWriter to pass to next.ServeHTTP and collect
		// downstream handler's response to query.  It will eventually be used
		// to flush to the client, assuming neither the handler panics, nor the
		// client connection is detected to be closed.
		grw := &responseWriter{hrw: w, begin: time.Now(), body: bb}

		if len(loggedHeaders) > 0 {
			// When any request headers are to be logged, this must copy the
			// respective values before it creates a go routine to handle
			// request.  Otherwise, if this must later time out the invoked
			// request header before that returns, that handler might
			// concurrently try to alter request headers while this is reading
			// them to emit the log line.
			grw.loggedRequestHeaders = make(map[string]string)
			for _, name := range loggedHeaders {
				value := r.Header.Get(name)
				if value == "" {
					value = "-"
				}
				grw.loggedRequestHeaders[name] = value // NOTE: only saves first value, because that is all that is logged.
			}
		}

		// Create a couple of channels to detect one of 3 ways to exit this
		// handler.
		handlerCompleted := make(chan struct{})
		handlerPanicked := make(chan string, 1)

		// We must invoke downstream handler in separate goroutine in order to
		// ensure this handler only responds to one of the three events below,
		// whichever event takes place first.
		go serveWithPanicProtection(grw, r, next, handlerCompleted, handlerPanicked)

		// Wait for the first of either of 3 events:
		//   * serveComplete: the next.ServeHTTP method completed normally
		//     (possibly even with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and
		//     panicked instead with a text message.
		//   * context is done: triggered when timeout or client disconnect.
		select {
		case <-handlerCompleted:
			grw.handlerComplete()
		case error := <-handlerPanicked:
			if config.AllowPanics {
				panic(error) // do not need to tell downstream to cancel, because it already panicked.
			}
			grw.handlerError(error, http.StatusInternalServerError)
		case <-ctx.Done():
			// While there are several reasons why the context may be closed,
			// when it is because the client terminates the request while the
			// handler is still working, the handler might still interact with
			// the response writer given to it.  Therefore, create and use a new
			// response writer for the remaining duration of this call that the
			// handler does not have simultaneous access to it.
			grw = &responseWriter{hrw: w, begin: grw.begin, body: bytes.NewBuffer(make([]byte, 0, 128))}

			// When the context is canceled, ctx.Err() will say why.  Returning
			// a 503 because this is what http.TimeoutHandler returns for same
			// case, even though this means 503 will be logged in server log
			// when client has terminated the connection.
			grw.handlerError(ctx.Err().Error(), http.StatusServiceUnavailable)
		}

		statusClass := grw.status / 100 // integer division (429 / 100 -> 4)

		// Update status counters
		if config.Counters != nil {
			atomic.AddUint64(&config.Counters.counters[0], 1)           // all
			atomic.AddUint64(&config.Counters.counters[statusClass], 1) // 1xx, 2xx, 3xx, 4xx, 5xx
		}

		// Invoke callback if provided, prior to logging request.
		var stats *Statistics
		if config.Callback != nil {
			stats = &Statistics{
				RequestBegin:   grw.begin,
				ResponseStatus: grw.status,
				ResponseEnd:    grw.end,
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
					emitter(grw, r, &buf)
				}
				_, _ = config.LogWriter.Write(buf)
			}
		}
	})
}
