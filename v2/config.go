package gohm

import (
	"bytes"
	"io"
	"time"
)

// Config specifies the parameters used for the wrapping the downstream
// http.Handler.
type Config struct {
	// AllowPanics, when set to true, causes panics to propagate from downstream
	// handlers.  When set to false, also the default value, panics will be
	// converted into Internal Server Errors (status code 500).  You cannot
	// change this setting after creating the http.Handler.
	AllowPanics bool

	// BufPool, when not nil, specifies a free-list pool of buffers to be used
	// to reduce garbage collection by reusing bytes.Buffer instances.
	BufPool BytesBufferPool

	// Callback, when not nil, is called after completion of each request with a
	// structure holding request and response information.
	Callback func(*Statistics)

	// Counters, if not nil, tracks counts of handler response status codes.
	Counters *Counters

	// EscrowReader specifies whether the middleware ought to provide an escrow
	// reader for the request body. The escrow reader reads the body exactly
	// once, storing the payload in a buffer, from which the downstream request
	// handler reads the data as it normally would, but also where an optional
	// Callback function might be able to have access to the request body
	// payload. When false, the specified Callback will return a nil for the
	// Statistics.ResponseBody.
	EscrowReader bool

	// LogBitmask, if not nil, specifies a bitmask to use to determine which
	// HTTP status classes ought to be logged.  If not set, all HTTP requests
	// will be logged.  This value may be changed using sync/atomic package even
	// after creating the http.Handler.
	//
	// The following bitmask values are supported:
	//
	//	LogStatus1xx    : LogStatus1xx used to log HTTP requests which have a 1xx response
	//	LogStatus2xx    : LogStatus2xx used to log HTTP requests which have a 2xx response
	//	LogStatus3xx    : LogStatus3xx used to log HTTP requests which have a 3xx response
	//	LogStatus4xx    : LogStatus4xx used to log HTTP requests which have a 4xx response
	//	LogStatus5xx    : LogStatus5xx used to log HTTP requests which have a 5xx response
	//	LogStatusAll    : LogStatusAll used to log all HTTP requests
	//	LogStatusErrors : LogStatusAll used to log HTTP requests which have 4xx or 5xx response
	LogBitmask *uint32

	// LogFormat specifies the format for log lines.  When left empty,
	// gohm.DefaultLogFormat is used.  You cannot change the log format after
	// creating the http.Handler.
	//
	// The following format directives are supported:
	//
	//	begin-epoch     : time request received (epoch)
	//	begin-iso8601   : time request received (ISO-8601 time format)
	//	begin           : time request received (apache log time format)
	//	bytes           : response size
	//	client-ip       : client IP address
	//	client-port     : client port
	//	client          : client-ip:client-port
	//	duration        : duration of request from beginning to end, (seconds with millisecond precision)
	//	end-epoch       : time request completed (epoch)
	//	end-iso8601     : time request completed (ISO-8601 time format)
	//	end             : time request completed (apache log time format)
	//	error           : error message associated with attempting to serve the query
	//	method          : request method, e.g., GET or POST
	//	proto           : request protocol, e.g., HTTP/1.1
	//	status          : response status code
	//	status-text     : response status text
	//	uri             : request URI
	LogFormat string

	// LogWriter, if not nil, specifies that log lines ought to be written to
	// the specified io.Writer.  You cannot change the io.Writer to which logs
	// are written after creating the http.Handler.
	LogWriter io.Writer

	// `Timeout`, when not 0, specifies the amount of time allotted to wait for
	// downstream `http.Handler` response.  You cannot change the handler
	// timeout after creating the `http.Handler`.  The zero value for Timeout
	// elides timeout protection, and `gohm` will wait forever for a downstream
	// `http.Handler` to return.  It is recommended that a sensible timeout
	// always be chosen for all production servers.
	Timeout time.Duration
}

// BytesBufferPool specifies any structure that can provide bytes.Buffer
// instances from a pool. One such performant and well-tested implementation for
// a free-list of buffers is https://github.com/karrick/gobp
type BytesBufferPool interface {
	Get() *bytes.Buffer
	Put(*bytes.Buffer)
}

// Statistics structures are passed to callback functions after the downstream
// handler has completed services the request.
type Statistics struct {
	// RequestBegin is the time the request handling began.
	RequestBegin time.Time

	// RequestBody is the byte slice of the request body, if applicable.
	RequestBody []byte

	// ResponseStatus is the status code of the response.
	ResponseStatus int

	// ResponseEnd is the time response writing completed.
	ResponseEnd time.Time
}
