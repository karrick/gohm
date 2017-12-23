package gohm

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type compressionResponseWriter struct {
	http.ResponseWriter
	compressionWriter io.Writer
}

func (g compressionResponseWriter) Write(b []byte) (int, error) {
	return g.compressionWriter.Write(b)
}

// WithGzip returns a new http.Handler that optionally compresses the response
// text using the gzip compression algorithm when the HTTP request's
// `Accept-Encoding` header includes the string `gzip`.
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
		defer func() {
			if err := gz.Close(); err != nil {
				Error(w, fmt.Sprintf("cannot compress stream: %s", err), http.StatusInternalServerError)
			}
		}()
		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(compressionResponseWriter{ResponseWriter: w, compressionWriter: gz}, r)
	})
}

// WithCompression returns a new http.Handler that optionally compresses the
// response text using either the gzip or deflate compression algorithm when the
// HTTP request's `Accept-Encoding` header includes the string `gzip` or
// `deflate`. To prevent the specified next http.Handler from also seeing the
// `Accept-Encoding` request header, and possibly also compressing the data a
// second time, this function removes that header from the request.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithCompression(someHandler))
func WithCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newWriteCloser io.WriteCloser
		var err error

		ae := r.Header.Get("Accept-Encoding")

		// Because many browsers include a buggy deflate compression algorithm,
		// prefer `gzip` over `deflate` if both are acceptable.
		if strings.Contains(ae, "gzip") {
			newWriteCloser = gzip.NewWriter(w)
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using gzip: %s", err), http.StatusInternalServerError)
				}
			}()
			w.Header().Set("Content-Encoding", "gzip")
		} else if strings.Contains(ae, "deflate") {
			newWriteCloser, err = flate.NewWriter(w, -1)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using deflate: %s", err), http.StatusInternalServerError)
				}
			}()
			w.Header().Set("Content-Encoding", "deflate")
		} else {
			next.ServeHTTP(w, r)
			return
		}
		r.Header.Del("Accept-Encoding")
		next.ServeHTTP(compressionResponseWriter{ResponseWriter: w, compressionWriter: newWriteCloser}, r)
	})
}
