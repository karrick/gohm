package gohm

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang/snappy"
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
		var err error
		var newWriteCloser io.WriteCloser
		var encodingName, responseHeaderName string

		requestHeaderName := "TE"
		acceptableEncodings := r.Header.Get(requestHeaderName)
		if acceptableEncodings != "" {
			// If transfer-encoding specified, then completely ignore
			// Accept-Encoding, because the upstream node has specifically
			// requested a node-to-node transfer compression algorithm.
			responseHeaderName = "Transfer-Encoding"
		} else {
			responseHeaderName = "Content-Encoding"
			requestHeaderName = "Accept-Encoding"
			acceptableEncodings = r.Header.Get(requestHeaderName)
		}

		// Because many browsers include a buggy deflate compression algorithm,
		// prefer `gzip` over `deflate` if both are acceptable.
		if encodingName = "snappy"; strings.Contains(acceptableEncodings, encodingName) {
			newWriteCloser = snappy.NewBufferedWriter(w)
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using snappy: %s", err), http.StatusInternalServerError)
				}
			}()
		} else if encodingName = "gzip"; strings.Contains(acceptableEncodings, encodingName) {
			newWriteCloser = gzip.NewWriter(w)
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using gzip: %s", err), http.StatusInternalServerError)
				}
			}()
		} else if encodingName = "deflate"; strings.Contains(acceptableEncodings, encodingName) {
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
		} else {
			next.ServeHTTP(w, r)
			return
		}
		r.Header.Del(requestHeaderName)
		w.Header().Set(responseHeaderName, encodingName)
		if responseHeaderName == "Content-Encoding" {
			w.Header().Set("Vary", responseHeaderName)
		}
		next.ServeHTTP(compressionResponseWriter{ResponseWriter: w, compressionWriter: newWriteCloser}, r)
	})
}
