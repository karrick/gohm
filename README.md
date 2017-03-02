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
    mux := http.NewServeMux()
    mux.Handle("/static/", gohm.LogErrors(os.Stderr, gohm.WithTimeout(30 * time.Second, gohm.ConvertPanicsToErrors(gohm.GzipHandler(someHandler))))
```

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because timeout handlers in Go are generally
implemented using a separate go routine, and the panic could occur in an alternate go routine and
not get caught by the `ConvertPanicsToErrors`.

## Supported Use Cases

### ConvertPanicsToErrors

ConvertPanicsToErrors returns a new http.Handler that catches all panics that may be caused by
the specified http.Handler, and responds with an appropriate http status code and message.

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because timeout handlers in Go are generally
implemented using a separate go routine, and the panic could occur in an alternate go routine and
not get caught by the `ConvertPanicsToErrors`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.ConvertPanicsToErrors(onlyGet(decodeURI(expand(querier)))))
```

### Error helper function

Error formats and emits the specified error message text and status code information to the
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
    mux.Handle("/example/path", gohm.LogAll(os.Stderr, decodeURI(expand(querier))))
```

### LogErrors

`LogErrors` returns a new `http.Handler` that logs HTTP requests that result in response errors, or
more specifically, HTTP status codes that are either 4xx or 5xx.  The handler will output lines in
common log format to the specified `io.Writer`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.LogErrors(os.Stderr, decodeURI(expand(querier))))
```

### Status1xxCounter

`Status1xxCounter` returns a new `http.Handler` that composes the specified next `http.Handler`, and
increments the specified counter when the response status code is not 1xx.

```Go
    var counter = expvar.NewInt("counter1xx")
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.Status1xxCounter(counter, decodeURI(expand(querier))))
```

### Status2xxCounter

`Status2xxCounter` returns a new `http.Handler` that composes the specified next `http.Handler`, and
increments the specified counter when the response status code is not 2xx.

```Go
    var counter = expvar.NewInt("counter2xx")
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.Status2xxCounter(counter, decodeURI(expand(querier))))
```

### Status3xxCounter

`Status3xxCounter` returns a new `http.Handler` that composes the specified next `http.Handler`, and
increments the specified counter when the response status code is not 3xx.

```Go
    var counter = expvar.NewInt("counter3xx")
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.Status3xxCounter(counter, decodeURI(expand(querier))))
```

### Status4xxCounter

`Status4xxCounter` returns a new `http.Handler` that composes the specified next `http.Handler`, and
increments the specified counter when the response status code is not 4xx.

```Go
    var counter = expvar.NewInt("counter4xx")
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.Status4xxCounter(counter, decodeURI(expand(querier))))
```

### Status5xxCounter

`Status5xxCounter` returns a new `http.Handler` that composes the specified next `http.Handler`, and
increments the specified counter when the response status code is not 5xx.

```Go
    var counter = expvar.NewInt("counter5xx")
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.Status5xxCounter(counter, decodeURI(expand(querier))))
```

### WithGzip

`WithGzip` returns a new `http.Handler` that optionally compresses the response text using the gzip
compression algorithm when the HTTP request's `Accept-Encoding` header includes the string `gzip`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.WithGzip(decodeURI(expand(querier))))
```

### WithTimeout

`WithTimeout` returns a new `http.Handler` that modifies creates a new `http.Request` instance with
the specified timeout set via context.

*NOTE:* When both the `WithTimeout` and the `ConvertPanicsToErrors` are used, the `WithTimeout`
ought to wrap the `ConvertPanicsToErrors`.  This is because timeout handlers in Go are generally
implemented using a separate go routine, and the panic could occur in an alternate go routine and
not get caught by the `ConvertPanicsToErrors`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.WithTimeout(30 * time.Second, onlyGet(decodeURI(expand(querier)))))
```
