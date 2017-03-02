package gohm

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter io.Writer
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gzipWriter.Write(b)
}

// WithGzip returns a new http.Handler that optionally compresses the response text using the
// gzip compression algorithm when the HTTP request's Accept-Encoding header includes the string
// "gzip".
//
// 	mux := http.NewServeMux()
// 	mux.Handle("/example/path", gohm.WithGzip(decodeURI(expand(querier))))
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
