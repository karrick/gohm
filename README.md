# gohm

gohm is a tiny Go library with HTTP middleware functions.

## Usage

Documentation is available via
[![GoDoc](https://godoc.org/github.com/karrick/gohm?status.svg)](https://godoc.org/github.com/karrick/gohm).

## Description

gohm provides a small collection of HTTP middleware functions to be used when creating a Go micro
webservice.

Here is a simple example:

```Go
const staticTimeout = time.Second // Used to control how long it takes to serve a static file.

var (
	// Will store statistics counters for status codes 1xx, 2xx, 3xx, 4xx, 5xx, as well as a
	// counter for all responses
	counters gohm.Counters

	// Used to dynamically control log level of HTTP logging.  After handler created, this must
	// be accessed using the sync/atomic package.
	logBitmask = gohm.LogStatusErrors

	// Determines HTTP log format
	logFormat = "{http-CLIENT-IP} {client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
)

func main() {

	h := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	h = gohm.WithGzip(h)                   // gzip response if client accepts gzip encoding
	h = gohm.WithTimeout(staticTimeout, h) // immediately return when downstream hasn't replied within specified time
	h = gohm.WithCloseNotifier(h)          // immediately return when the client disconnects
	h = gohm.ConvertPanicsToErrors(h)      // when downstream panics, convert to 500
	h = gohm.StatusCounters(&counters, h)  // update counter stats for 1xx, 2xx, 3xx, 4xx, 5xx, and all queries
	h = gohm.LogStatusBitmaskWithFormat(logFormat, &logBitmask, os.Stderr, h)

	http.Handle("/static/", h)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

In the above example notice that each successive line wraps the handler of the line above it.  The
following two examples are equivalent, and both involve handlerA to perform some setup then invoking
handlerB, which performs its setup work, and finally invokes handlerA.  Both do the same thing, but
source code looks vastly different.  In both cases, handlerA is considered upstream from handlerB,
which is considered upstream of handlerC.  Similarly, handlerC is downstream of handlerB, which is
likewise downstream of handlerA.

In the example code, the format from example2 was used because it keeps source code lines from
getting too long.

```Go
func example1() {
	h := handlerA(handlerB(handlerC))
}

func example2() {
	h := handlerC
	h = handlerB(h)
	h = handlerA(h)
}
```

While not all of these handlers are required, the order may be important in some circumstances.  For
instance, if `gohm.ConvertPanicsToErrors` is between `gohm.StatisCounters` and `gohm.Log...`, then a
panic will not cause the 5xx status counter to be incremented.

## Helper Functions

### Error

`Error` formats and emits the specified error message text and status code information to the
`http.ResponseWriter`, to be consumed by the client of the service.  This particular helper function
has nothing to do with emitting log messages on the server side, and only creates a response for the
client.  Typically handlers will call this method prior to invoking return to return to whichever
handler invoked it.

```Go
// example function which guards downstream handlers to ensure only HTTP GET method used to
// access resource.
func onlyGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			gohm.Error(w, r.Method, http.StatusMethodNotAllowed)
			// 405 Method Not Allowed: POST
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

## HTTP Handler Middleware Functions

### ConvertPanicsToErrors

`ConvertPanicsToErrors` returns a new `http.Handler` that catches all panics that may be caused by
the specified `http.Handler`, and responds with an appropriate HTTP status code and message.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.ConvertPanicsToErrors(onlyGet(someHandler)))
```

### LogAll

`LogAll` returns a new `http.Handler` that logs HTTP requests and responses using the
`gohm.DefaultLogFormat` to the specified `io.Writer`.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.LogAll(os.Stderr, someHandler))
```

### LogAllWithFormat

`LogAllWithFormat` returns a new `http.Handler` that logs HTTP requests and responses using the
specified log format string to the specified `io.Writer`.

```Go
mux := http.NewServeMux()
format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
mux.Handle("/example/path", gohm.LogAllWithFormat(format, os.Stderr, someHandler))
```

### LogErrors

`LogErrors` returns a new `http.Handler` that logs HTTP requests that result in response errors, or
more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output lines
using the `gohm.DefaultLogFormat` to the specified `io.Writer`.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.LogErrors(os.Stderr, someHandler))
```

### LogErrorsWithFormat

`LogErrorsWithFormat` returns a new `http.Handler` that logs HTTP requests that result in response
errors, or more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output
lines using the specified log format string to the specified `io.Writer`.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.LogErrors(os.Stderr, someHandler))
```

### LogStatusBitmask

`LogStatusBitmask` returns a new `http.Handler` that logs HTTP requests that have a status code that
matches any of the status codes in the specified bitmask.  The handler will output lines using the
`gohm.DefaultLogFormat` to the specified `io.Writer`.

The `bitmask` parameter is used to specify which HTTP requests ought to be logged based on the HTTP
status code returned by the next `http.Handler`.

```Go
mux := http.NewServeMux()
logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
mux.Handle("/example/path", gohm.LogStatusBitmask(&logBitmask, os.Stderr, someHandler))
```

### LogStatusBitmaskWithFormat

`LogStatusBitmaskWithFormat` returns a new `http.Handler` that logs HTTP requests that have a status
code that matches any of the status codes in the specified bitmask.  The handler will output lines
in the specified log format string to the specified `io.Writer`.

The following format directives are supported:

	begin-epoch:     time request received (epoch)
	begin-iso8601:   time request received (ISO-8601 time format)
	begin:           time request received (apache log time format)
	bytes:           response size
	client-ip:       client IP address
	client-port:     client port
	client:          client-ip:client-port
	duration:        duration of request from beginning to end, (seconds with millisecond precision)
	end-epoch:       time request completed (epoch)
	end-iso8601:     time request completed (ISO-8601 time format)
	end:             time request completed (apache log time format)
	method:          request method, e.g., GET or POST
	proto:           request protocol, e.g., HTTP/1.1
	status:          response status code
	uri:             request URI

In addition, values from HTTP request headers can also be included in the log by prefixing the HTTP
header name with `http-`.  In the below example, each log line will begin with the value of the HTTP
request header `CLIENT-IP`:

```Go
format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
```

The `bitmask` parameter is used to specify which HTTP requests ought to be logged based on the HTTP
status code returned by the next `http.Handler`.

```Go
mux := http.NewServeMux()
format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
mux.Handle("/example/path", gohm.LogStatusBitmaskWithFormat(format, &logBitmask, os.Stderr, someHandler))
```

### StatusCounters

`StatusCounters` returns a new `http.Handler` that increments the specified `gohm.Counters` for
every HTTP response based on the status code of the specified `http.Handler`.

```Go
var counters gohm.Counters
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.StatusCounters(&counters, someHandler))
// later on...
status1xxCounter := counters.Get1xx()
```

### WithCloseNotifier

`WithCloseNotifier` returns a new `http.Handler` that attempts to detect when the client has closed
the connection, and if it does so, immediately returns with an appropriate error message to be
logged, while sending a signal to context-aware downstream handlers.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.WithCloseNotifier(someHandler))
```

### WithGzip

`WithGzip` returns a new `http.Handler` that optionally compresses the response text using the gzip
compression algorithm when the HTTP request's `Accept-Encoding` header includes the string `gzip`.

```Go
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.WithGzip(someHandler))
```

### WithTimeout

`WithTimeout` returns a new `http.Handler` that creates a watchdog goroutine to detect when the
timeout has expired.  It also modifies the request to add a context timeout, because while not all
handlers use context and respect context timeouts, it's likely that more and more will over time as
context becomes more popular.

Unlike when using `http.TimeoutHandler`, if a downstream `http.Handler` panics, this handler will
catch that panic in the other goroutine and re-play it in the primary goroutine, allowing upstream
handlers to catch the panic if desired.  Panics may be caught by the `gohm.ConvertPanicsToErrors`
handler when placed upstream of this handler.

```Go
mux := http.NewServeMux()
mux.Handle("/example/path", gohm.WithTimeout(10 * time.Second, someHandler))
```
