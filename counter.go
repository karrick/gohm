package gohm

import (
	"expvar"
	"net/http"
)

type counterHandler struct {
	http.ResponseWriter
	status int
}

func (r *counterHandler) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func member(status, group int) bool {
	return status/group == 1 && status%group < 100
}

// StatusAllCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter for every query.
//
//	var counterAll = expvar.NewInt("counterAll")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.StatusAllCounter(counterAll, decodeURI(expand(querier))))
func StatusAllCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		counter.Add(1)
	})
}

// Status1xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 1xx.
//
//	var counter1xx = expvar.NewInt("counter1xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status1xxCounter(counter1xx, decodeURI(expand(querier))))
func Status1xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if !member(ch.status, 100) {
			counter.Add(1)
		}
	})
}

// Status2xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 2xx.
//
//	var counter2xx = expvar.NewInt("counter2xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status2xxCounter(counter2xx, decodeURI(expand(querier))))
func Status2xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		// NOTE: Also need to check for zero-value of status variable, because when omitted
		// by handler, it's filled in later in http stack.
		if ch.status != 0 && !member(ch.status, 200) {
			counter.Add(1)
		}
	})
}

// Status3xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 3xx.
//
//	var counter3xx = expvar.NewInt("counter3xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status3xxCounter(counter3xx, decodeURI(expand(querier))))
func Status3xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if !member(ch.status, 300) {
			counter.Add(1)
		}
	})
}

// Status4xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 4xx.
//
//	var counter4xx = expvar.NewInt("counter4xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status4xxCounter(counter4xx, decodeURI(expand(querier))))
func Status4xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if !member(ch.status, 400) {
			counter.Add(1)
		}
	})
}

// Status5xxCounter returns a new http.Handler that composes the specified next http.Handler,
// and increments the specified counter when the response status code is not 5xx.
//
//	var counter5xx = expvar.NewInt("counter5xx")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.Status5xxCounter(counter5xx, decodeURI(expand(querier))))
func Status5xxCounter(counter *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := &counterHandler{ResponseWriter: w}
		next.ServeHTTP(ch, r)
		if !member(ch.status, 500) {
			counter.Add(1)
		}
	})
}
