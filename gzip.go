package gohm

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter    io.Writer
	closeNotifyCh <-chan bool
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gzipWriter.Write(b)
}

// WithGzip returns a new http.Handler that optionally compresses the response text using the gzip
// compression algorithm when the HTTP request's Accept-Encoding header includes the string "gzip".
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithGzip(someHandler))
func WithGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzip.NewWriter(w)
		defer func() { _ = gz.Close() }()
		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipResponseWriter{ResponseWriter: w, gzipWriter: gz}, r)
	})
}

func (r *gzipResponseWriter) CloseNotify() <-chan bool {
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
