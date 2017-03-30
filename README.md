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
	var (
	    globalTimeout = 10 * time.Second
	    status1xxCounter *expvar.Int
	    // and so on
	)
	
	func init() {
	    status1xxCount = expvar.NewInt()
	    // and so on
	}
	
	func main() {
	    var counters gohm.Counters
	
	    var h http.Handler = http.NewServeMux()
		h = gohm.WithGzip(h)
		h = gohm.ConvertPanicsToErrors(h)
		h = gohm.WithTimeout(10 * time.Second, h)
		h = gohm.StatusCounters(&counters, h)
	
	    format := "{http-CLIENT-IP} {client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
	    logBitmask := uint32(gohm.LogStatus4xx|gohm.LogStatus5xx)
		h = gohm.LogStatusBitmaskPreCompile(format, &globalLogBitmask, logs, h)
	
	    mux.Handle("/static/", h)
	}
```

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because `WithTimeout` does not itself implement
the timeout, but requests the `net/http` library to do so, which implements timeout handling using a
separate go routine.  When a panic occurs in a separate go routine it will not get caught by
`ConvertPanicsToErrors`.

## Supported Use Cases

### ConvertPanicsToErrors

`ConvertPanicsToErrors` returns a new `http.Handler` that catches all panics that may be caused by
the specified `http.Handler`, and responds with an appropriate HTTP status code and message.

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because `WithTimeout` does not itself implement
the timeout, but requests the `net/http` library to do so, which implements timeout handling using a
separate go routine.  When a panic occurs in a separate go routine it will not get caught by
`ConvertPanicsToErrors`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.ConvertPanicsToErrors(onlyGet(someHandler)))
```

### Error helper function

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

### LogAll

`LogAll` returns a new `http.Handler` that logs HTTP requests and responses in common log format to
the specified `io.Writer`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.LogAll(os.Stderr, someHandler))
```

### LogErrors

`LogErrors` returns a new `http.Handler` that logs HTTP requests that result in response errors, or
more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output lines in
common log format to the specified `io.Writer`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.LogErrors(os.Stderr, someHandler))
```

### LogStatusBitmask

`LogStatusBitmask` returns a new `http.Handler` that logs HTTP requests that have a status code that
matches any of the status codes in the specified bitmask.  The handler will output lines in common
log format to the specified `io.Writer`.

The `bitmask` parameter is used to specify which HTTP requests ought to be logged based on the HTTP
status code returned by the next `http.Handler`.

The default log format line is:

     "{client-ip} - [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"

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

	begin:           time request received (apache log time format)
	begin-epoch:     time request received (epoch)
	begin-iso8601:   time request received (ISO-8601 time format)
	bytes:           response size
	client:          client-ip:client-port
	client-ip:       client IP address
	client-port:     client port
	duration:        duration of request from begin to end, (seconds with millisecond precision)
	end:             time request completed (apache log time format)
	end-epoch:       time request completed (epoch)
	end-iso8601:     time request completed (ISO-8601 time format)
	method:          request method, e.g., GET or POST
	proto:           request protocol, e.g., HTTP/1.1
	status:          response status code
	uri:             request URI

In addition, values from HTTP request headers can also be included in the log by prefixing the HTTP
header name with `http-`.  In the below example, each log line will start with the value of the HTTP
request header `CLIENT-IP`:

     format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"

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

### WithGzip

`WithGzip` returns a new `http.Handler` that optionally compresses the response text using the gzip
compression algorithm when the HTTP request's `Accept-Encoding` header includes the string `gzip`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.WithGzip(someHandler))
```

### WithTimeout

`WithTimeout` returns a new `http.Handler` that modifies creates a new `http.Request` instance with
the specified timeout set via context.

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because `WithTimeout` does not itself implement
the timeout, but requests the `net/http` library to do so, which implements timeout handling using a
separate go routine.  When a panic occurs in a separate go routine it will not get caught by
`ConvertPanicsToErrors`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.WithTimeout(30 * time.Second, onlyGet(someHandler)))
```
