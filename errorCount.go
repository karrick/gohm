package gohm

import (
	"expvar"
	"net/http"
)

// ErrorCountHandler returns a new http.Handler that composes the specified next http.Handler, and
// increments the specified counter when the response status code is not http.StatusOK.
//
//	var errorCount = expvar.NewInt("/example/path/errorCount")
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.ErrorCount(errorCount, decodeURI(expand(querier))))
func ErrorCountHandler(errorCount *expvar.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eh := &errorCountHandler{ResponseWriter: w}
		next.ServeHTTP(eh, r)
		// NOTE: check for status zero value because when omitted by handler, it's filled in later in http stack
		if eh.status != http.StatusOK && eh.status != 0 {
			errorCount.Add(1)
		}
	})
}

type errorCountHandler struct {
	http.ResponseWriter
	status int
}

func (r *errorCountHandler) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
