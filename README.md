# gohm

gohm is a tiny Go library with HTTP middleware functions.

## Usage

Documentation is available via
[![GoDoc](https://godoc.org/github.com/karrick/gohm?status.svg)](https://godoc.org/github.com/karrick/gohm).

### Versions

It is customary to use semantic versioning to tag project releases. This library
received a v2.x.y tag one year ago today--as this comment is being written--and
there are a few packages which depend on v2.x.y version of this package.

Recently a number of blog postings by Russ Cox proposed and described a version
aware `go build` tool-chain, where one of its requirements was that in order to
maintain version 1 compatibility for upstream users of a library, source code
for version 1 of a project ought to remain in the top-level of the package
repository. Furthermore, version 2 of a library ought to be placed in its own
package by having its source code located in a `v2/` subdirectory from the
top-level of the project's repository.

In order to support other projects with a pinned dependency that happens to have
this package as a transitive dependency, yet does not itself pin this package
version, a snapshot of today's source code for version 2 of this project is made
available in the top-level of the repository. Additional development work in the
project will continue in the `v2/` subdirectory, and users are encouraged to
import that version in their projects:

#### To use the newest features of this library

```Go
import gohm "github.com/karrick/gohm/v2"
```

#### To use the features of this library available as of 2018-04-04

```Go
import "github.com/karrick/gohm"
```

#### To use version 1 of this library

```Go
import gohm "gopkg.in/karrick/gohm.v1"
```

## Description

`gohm` provides a small collection of HTTP middleware functions to be used when
creating a Go micro webservice.  With the exception of handler timeout control,
all of the configuration options have sensible defaults, so an empty
`gohm.Config{}` object may be used to initialize the `http.Handler` wrapper to
start, and further customization is possible down the road.  Using the default
handler timeout elides timeout protection, so it's recommended that timeouts are
always created for production code.

Here is a simple example:

```Go
package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"
    "path/filepath"

    gohm "github.com/karrick/gohm/v2"
)

func main() {
    optPort := flag.Int("port", 8080, "HTTP server network port")
    optStatic := flag.String("static", "static", "filesystem pathname to static virtual root")
    flag.Parse()

    *optStatic = filepath.Clean(*optStatic)

    // create mux rather than using http.DefaultServeMux so we can later wrap it
    // with gohm.New to provide logging, error handling, along with panic and
    // timeout protection.
    mux := http.NewServeMux()

    // static resources
    mux.Handle("/static/", gohm.StaticHandler("/static/", *optStatic))

    // default handler serves index page for empty URI, "/", but 404 for
    // everything else.
    mux.Handle("/", gohm.DefaultHandler(filepath.Join(*optStatic, "index.html")))

    log.Print("[INFO] web service port: ", *optPort)
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", *optPort),
        Handler: gohm.New(gohm.WithCompression(mux), gohm.Config{Timeout: time.Second}),
    }

    if err := server.ListenAndServe(); err != nil {
        log.Fatal("[ERROR] ", err)
    }
}
```

Here is an example with a few customizations:

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
    logFormat = "{http-CLIENT-IP} {client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration} {message}"
)

func main() {
    optStatic := flag.String("static", "static", "filesystem pathname to static virtual root")
    flag.Parse()

    *optStatic = filepath.Clean(*optStatic)

    h := gohm.StaticHandler("/static/", *optStatic)
    h = gohm.WithCompression(h)

    // gohm was designed to wrap other http.Handler functions.
    h = gohm.New(h, gohm.Config{
        Counters:   &counters,   // pointer given so counters can be collected and optionally reset
        LogBitmask: &logBitmask, // pointer given so bitmask can be updated using sync/atomic
        LogFormat:  logFormat,
        LogWriter:  os.Stderr,
        Timeout:    staticTimeout,
    })

    http.Handle("/static/", h)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

In the above example notice that each successive line wraps the handler of the
line above it.  The terms upstream and downstream do not refer to which line
was above which other line in the source code.  Rather, upstream handlers
invoke downstream handlers.  In both of the above examples, the top level
handler is `gohm`, which is upstream of `gohm.WithGzip`, which in turn is
upstream of `http.StripPrefix`, which itself is upstream of `http.FileServer`,
which finally is upstream of `http.Dir`.

As another illustration, the following two example functions are equivalent,
and both invoke `handlerA` to perform some setup then invoke `handlerB`, which
performs its setup work, and finally invokes `handlerC`.  Both do the same
thing, but source code looks vastly different.  In both cases, `handlerA` is
considered upstream from `handlerB`, which is considered upstream of
`handlerC`.  Similarly, `handlerC` is downstream of `handlerB`, which is
likewise downstream of `handlerA`.

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

Sometimes it is necessary to cast a regular function to the http.HandleFunc
type, as shown below.

```Go
func fooHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello, World!\r\n"))
}

func example() {
    h := gohm.WithCompression(http.HandlerFunc(fooHandler))
    // ...
}
```

## Helper Functions

### Error

`Error` formats and emits the specified error message text and status code
information to the `http.ResponseWriter`, to be consumed by the client of the
service.  This particular helper function has nothing to do with emitting log
messages on the server side, and only creates a response for the client.
However, if a handler that invokes `gohm.Error` is wrapped with logging
functionality by `gohm.New`, then `gohm` will also emit a sensible log message
based on the specified status code and message text.  Typically handlers will
call this method prior to invoking return to return to whichever handler invoked
it.

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

### New

`New` returns a new `http.Handler` that calls the specified next `http.Handler`,
and performs the requested operations before and after the downstream handler as
specified by the `gohm.Config` structure passed to it.

It receives a `gohm.Config` instance rather than a pointer to one, to discourage
modification after creating the `http.Handler`.  With the exception of handler
timeout control, all of the configuration options have sensible defaults, so an
empty `gohm.Config{}` object may be used to initialize the `http.Handler`
wrapper to start, and further customization is possible down the road.  Using
the default handler timeout elides timeout protection, so it's recommended that
timeouts are always created for production code.  Documentation of the
`gohm.Config` structure provides additional details for the supported
configuration fields.

#### Configuration Parameters

##### AllowPanics

`AllowPanics`, when set to true, causes panics to propagate from downstream
handlers.  When set to false, also the default value, panics will be converted
into Internal Server Errors (status code 500).  You cannot change this setting
after creating the `http.Handler`.

##### BufPool

`BufPool`, when not nil, specifies a free-list pool of buffers to be used to
reduce garbage collection by reusing bytes.Buffer instances.

When BufPool is non-nil and EscrowReader is true, and a request is received by
gohm which has been configured to read in the payload before calling the
handler, the provided BufPool's Get method will be invoked to obtain a
bytes.Buffer to hold the bytes from the request payload. After the request
handler has completed the bytes.Buffer will be returned to the specified
BufPool by invoking its Put method.

See `examples/payload/main.go` for an example of using BufPool, Callback, and
EscrowReader.

##### Callback

By default after the request handler has completed its work, gohm will
optionally log request statistics prior to releasing resources. Sometimes an
application needs to perform some post-request operations, and may do so in the
specified Callback function. The Callback function is invoked with a Statistics
argument that provides the begin and end times of the request, a slice of bytes
provided in the request body, and the numeric response status code. The
provided Statistics structure provides a nilary Log method that forces gohm to
emit a log of the request regardless of any other logging flags.

See `examples/payload/main.go` for an example of using BufPool, Callback, and
EscrowReader.

##### Counters

`Counters`, if not nil, tracks counts of handler response status codes.

##### EscrowReader

By default request handlers will read the request payload from the
http.Request's Body field, an io.ReadCloser that the request handler is
responsible to close. However, some handlers may want to re-read the payload,
or sometimes a provided Callback method needs to reprocess the request payload
after the handler has completed. When EscrowReader is set to true, gohm will
fully consume the request body payload, storing it in a buffer, and allowing
no-penalty reads and re-reads. It ensures the original handler may continue to
read and close the provided http.Request's Body field as normal, to minimize
code change to handlers that are not aware of the optimization.

See `examples/payload/main.go` for an example of using BufPool, Callback, and
EscrowReader.

##### LogBitmask

The `LogBitmask` parameter is used to specify which HTTP requests ought to be
logged based on the HTTP status code returned by the downstream `http.Handler`.

##### LogFormat

The following format directives are supported.  All times provided are converted
to UTC before formatting.

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
    error:           context timeout, context closed, or panic error message
    method:          request method, e.g., GET or POST
    proto:           request protocol, e.g., HTTP/1.1
    status:          response status code
    status-text:     response status text
    uri:             request URI

In addition, values from HTTP request headers can also be included in the log by
prefixing the HTTP header name with `http-`.  In the below example, each log
line will begin with the value of the HTTP request header `CLIENT-IP`.  If the
specified request header is not present, a hyphen will be used in place of the
non-existant value.

```Go
format := "{http-CLIENT-IP} {http-USER} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
```

##### LogWriter

`LogWriter`, if not nil, specifies that log lines ought to be written to the
specified `io.Writer`.  You cannot change the `io.Writer` to which logs are
written after creating the `http.Handler`.

##### Timeout

`Timeout`, when not 0, specifies the amount of time allotted to wait for
downstream `http.Handler` response.  You cannot change the handler timeout after
creating the `http.Handler`.  The zero value for Timeout elides timeout
protection, and `gohm` will wait forever for a downstream `http.Handler` to
return.  It is recommended that a sensible timeout always be chosen for all
production servers.

### WithCompression

`WithCompression` returns a new `http.Handler` that optionally compresses the
response text using the gzip or deflate compression algorithm when the HTTP
request's `Accept-Encoding` header includes the string `gzip` or `deflate`.

```Go
    mux := http.NewServeMux()
    staticPath := "static"
    mux.Handle("/static/", gohm.WithCompression(gohm.StaticHandler("/static/", staticPath)))
```

### WithGzip

`WithGzip` returns a new `http.Handler` that optionally compresses the response
text using the gzip compression algorithm when the HTTP request's
`Accept-Encoding` header includes the string `gzip`.

```Go
    mux := http.NewServeMux()
    mux.Handle("/example/path", gohm.WithGzip(someHandler))
```
