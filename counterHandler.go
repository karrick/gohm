package gohm

import (
	"expvar"
	"net/http"
)

// Status1xxCounterHandler returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 1xx.
//
//	var counter1xx = expvar.NewInt("counter1xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status1xxConterHandler(counter1xx, decodeURI(expand(querier))))
func Status1xxCounterHandler(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		if eh.status/100 != 1 {
			counter.Add(1)
		}
	})
}

// Status2xxCounterHandler returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 2xx.
//
//	var counter2xx = expvar.NewInt("counter2xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status2xxConterHandler(counter2xx, decodeURI(expand(querier))))
func Status2xxCounterHandler(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		// NOTE: Also need to check for zero value of status variable, because when omitted
		// by handler, it's filled in later in http stack.
		if eh.status != 0 && eh.status/200 != 1 {
			counter.Add(1)
		}
	})
}

// Status3xxCounterHandler returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 3xx.
//
//	var counter3xx = expvar.NewInt("counter3xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status3xxConterHandler(counter3xx, decodeURI(expand(querier))))
func Status3xxCounterHandler(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		if eh.status/300 != 1 {
			counter.Add(1)
		}
	})
}

// Status4xxCounterHandler returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 4xx.
//
//	var counter4xx = expvar.NewInt("counter4xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status4xxConterHandler(counter4xx, decodeURI(expand(querier))))
func Status4xxCounterHandler(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		if eh.status/400 != 1 {
			counter.Add(1)
		}
	})
}

// Status5xxCounterHandler returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 5xx.
//
//	var counter5xx = expvar.NewInt("counter5xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status5xxConterHandler(counter5xx, decodeURI(expand(querier))))
func Status5xxCounterHandler(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		if eh.status/500 != 1 {
			counter.Add(1)
		}
	})
}

type counterHandler struct {
	http.ResponseWriter
	status int
}

func (r *counterHandler) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
