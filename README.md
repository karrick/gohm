# gohm

gohm is a tiny Go library with HTTP middleware functions.

## Usage

Documentation is available via
[![GoDoc](https://godoc.org/github.com/karrick/gohm?status.svg)](https://godoc.org/github.com/karrick/gohm).

## Description

gohm provides a small collection of HTTP middleware functions to be used when creating a Go micro
webservice.

## Supported Use Cases

### Error helper function

Error formats and emits the specified error message text and status code information to the
http.ResponseWriter, to be consumed by the client of the service.  This particular helper
function has nothing to do with emitting log messages on the server side, and only creates a
response for the client.  Typically handlers will call this method prior to invoking return to
exit to whichever handler invoked it.

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

### ErrorCount

ErrorCountHandler returns a new http.Handler that composes the specified next http.Handler, and
increments the specified counter when the response status code is not http.StatusOK.

```Go
	var errorCount = expvar.NewInt("/example/path/errorCount")
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.ErrorCount(errorCount, decodeURI(expand(querier))))
```

### ErrorLogHandler

ErrorLogHandler returns a new http.Handler that logs HTTP requests that result in response
errors. The handler will output lines in common log format to the specified io.Writer.

### GzipHandler

GzipHandler returns a new http.Handler that optionally compresses the response text using the gzip
compression algorithm when the HTTP request's Accept-Encoding header includes the string "gzip".

```Go
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.GzipHandler(decodeURI(expand(querier))))
```

### LogHandler

LogHandler returns a new http.Handler that logs HTTP requests and responses in common log format to
the specified io.Writer.

```Go
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.LogHandler(os.Stderr, decodeURI(expand(querier))))
```

### PanicHandler

Rather than dumping a stack trace and exiting when a downstream HTTP handler panics, PanicHandler
recovers from the panic and converts the event into an standard HTTP error event, suitable for
logging upstream if desired.

```Go
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.PanicHandler(onlyGet(decodeURI(expand(querier)))))
```

### TimeoutHandler

TimeoutHandler returns a new http.Handler that modifies creates a new http.Request instance with the
specified timeout set via context.

```Go
	mux := http.NewServeMux()
	mux.Handle("/example/path", gohm.TimeoutHandler(30 * time.Second, onlyGet(decodeURI(expand(querier)))))
```
