package gohm

import (
	"expvar"
	"net/http"
	"sync/atomic"
)

type countingResponseWriter struct {
	http.ResponseWriter
	status        int
	closeNotifyCh <-chan bool
}

func (r *countingResponseWriter) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *countingResponseWriter) CloseNotify() <-chan bool {
	if r.closeNotifyCh != nil {
		return r.closeNotifyCh
	}
	if notifier, ok := r.ResponseWriter.(http.CloseNotifier); ok {
		r.closeNotifyCh = notifier.CloseNotify()
	} else {
		// Return a channel that nothing will ever emit to, and will eventually be garbage
		// collected.
		//
		// NOTE: I am not absolutely certain about the client side-effects of essentially
		// broadcasting this as a CloseNotifier when it returns a dummy channel.  Well
		// behaved http.Handler functions will attempt to take advantage of the feature, but
		// it will sadly not work.  This can happen when a server program inserts a
		// http.Handler into the pipeline for a call that inserts its own
		// http.ResponseHandler that does not have the CloseNotify method.
		r.closeNotifyCh = make(<-chan bool)
	}
	return r.closeNotifyCh
}

// StatusCounters returns a new http.Handler that increments the specified gohm.Counters for every
// HTTP response based on the status code of the specified http.Handler.
//
//	var counters gohm.Counters
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.StatusCounters(&counters, someHandler))
//	// later on...
//	status1xxCounter := counters.Get1xx()
func StatusCounters(counters *Counters, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(ch, r)
		atomic.AddUint64(&counters.counterAll, 1)
		switch ch.status / 100 {
		case 1:
			atomic.AddUint64(&counters.counter1xx, 1)
		case 2:
			atomic.AddUint64(&counters.counter2xx, 1)
		case 3:
			atomic.AddUint64(&counters.counter3xx, 1)
		case 4:
			atomic.AddUint64(&counters.counter4xx, 1)
		case 5:
			atomic.AddUint64(&counters.counter5xx, 1)
		}
	})
}

// Counters structure stores status counters used to track number of HTTP responses resulted in
// various status codes.  The counts are grouped by the status code groups.
//
//	var counters gohm.Counters
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.StatusCounters(&counters, someHandler))
//	// later on...
//	countOf1xx := counters.Get1xx()
//	countOf2xx := counters.Get2xx()
//	countOf3xx := counters.Get3xx()
//	countOf4xx := counters.Get4xx()
//	countOf5xx := counters.Get5xx()
//	countTotal := counters.GetAll()
type Counters struct {
	counterAll, counter1xx, counter2xx, counter3xx, counter4xx, counter5xx uint64
}

// Get1xx returns number of HTTP responses resulting in a 1xx status code.
func (c Counters) Get1xx() uint64 {
	return atomic.LoadUint64(&c.counter1xx)
}

// Get2xx returns number of HTTP responses resulting in a 2xx status code.
func (c Counters) Get2xx() uint64 {
	return atomic.LoadUint64(&c.counter2xx)
}

// Get3xx returns number of HTTP responses resulting in a 3xx status code.
func (c Counters) Get3xx() uint64 {
	return atomic.LoadUint64(&c.counter3xx)
}

// Get4xx returns number of HTTP responses resulting in a 4xx status code.
func (c Counters) Get4xx() uint64 {
	return atomic.LoadUint64(&c.counter4xx)
}

// Get5xx returns number of HTTP responses resulting in a 5xx status code.
func (c Counters) Get5xx() uint64 {
	return atomic.LoadUint64(&c.counter5xx)
}

// GetAll returns total number of HTTP responses, regardless of status code.
func (c Counters) GetAll() uint64 {
	return atomic.LoadUint64(&c.counterAll)
}

// GetAndReset1xx returns number of HTTP responses resulting in a 1xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset1xx() uint64 {
	return atomic.SwapUint64(&c.counter1xx, 0)
}

// GetAndReset2xx returns number of HTTP responses resulting in a 2xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset2xx() uint64 {
	return atomic.SwapUint64(&c.counter2xx, 0)
}

// GetAndReset3xx returns number of HTTP responses resulting in a 3xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset3xx() uint64 {
	return atomic.SwapUint64(&c.counter3xx, 0)
}

// GetAndReset4xx returns number of HTTP responses resulting in a 4xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset4xx() uint64 {
	return atomic.SwapUint64(&c.counter4xx, 0)
}

// GetAndReset5xx returns number of HTTP responses resulting in a 5xx status code, and resets the
// counter to 0.
func (c Counters) GetAndReset5xx() uint64 {
	return atomic.SwapUint64(&c.counter5xx, 0)
}

// GetAndResetAll returns number of HTTP responses resulting in a All status code, and resets the
// counter to 0.
func (c Counters) GetAndResetAll() uint64 {
	return atomic.SwapUint64(&c.counterAll, 0)
}

// StatusAllCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter for every query.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counterAll = expvar.NewInt("counterAll")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.StatusAllCounter(counterAll, someHandler))
func StatusAllCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		counter.Add(1)
	})
}

// Status1xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 1xx.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counter1xx = expvar.NewInt("counter1xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status1xxCounter(counter1xx, someHandler))
func Status1xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if ch.status/100 == 1 {
			counter.Add(1)
		}
	})
}

// Status2xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 2xx.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counter2xx = expvar.NewInt("counter2xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status2xxCounter(counter2xx, someHandler))
func Status2xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		// NOTE: Also need to check for zero-value of status variable, because when omitted
		// by handler, it's filled in later in http stack.
		if ch.status == 0 || ch.status/100 == 2 {
			counter.Add(1)
		}
	})
}

// Status3xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 3xx.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counter3xx = expvar.NewInt("counter3xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status3xxCounter(counter3xx, someHandler))
func Status3xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if ch.status/100 == 3 {
			counter.Add(1)
		}
	})
}

// Status4xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 4xx.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counter4xx = expvar.NewInt("counter4xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status4xxCounter(counter4xx, someHandler))
func Status4xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if ch.status/100 == 4 {
			counter.Add(1)
		}
	})
}

// Status5xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 5xx.
//
// Deprecated: Use gohm.StatusCounters instead.
//
//	var counter5xx = expvar.NewInt("counter5xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status5xxCounter(counter5xx, someHandler))
func Status5xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if ch.status/100 == 5 {
			counter.Add(1)
		}
	})
}
