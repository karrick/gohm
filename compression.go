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
// NOTE: The specified next http.Handler ought not set `Content-Length` header,
// or the reported length value will be wrong. As a matter of fact, all HTTP
// response handlers ought to allow net/http library to set `Content-Length`
// response header or not based on a handful of RFCs.
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
		w.Header().Set("Vary", "Accept-Encoding")
		next.ServeHTTP(compressionResponseWriter{ResponseWriter: w, compressionWriter: gz}, r)
	})
}

// WithCompression returns a new http.Handler that optionally compresses the
// response text using either the gzip or deflate compression algorithm when the
// HTTP request's `Accept-Encoding` header includes the string `gzip` or
// `deflate`. To prevent the downstream http.Handler from also seeing the
// `Accept-Encoding` request header, and possibly also compressing the data a
// second time, this function removes that header from the request.
//
// NOTE: The specified next http.Handler ought not set `Content-Length` header,
// or the reported length value will be wrong. As a matter of fact, all HTTP
// response handlers ought to allow net/http library to set `Content-Length`
// response header or not based on a handful of RFCs.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithCompression(someHandler))
func WithCompression(next http.Handler) http.Handler {
	const requestHeader = "Accept-Encoding"
	const responseHeader = "Content-Encoding"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newWriteCloser io.WriteCloser
		var encodingAlgorithm string

		acceptableEncodings := r.Header.Get(requestHeader)

		// Shortcut if no Accept-Encoding header
		if acceptableEncodings == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Offer gzip, and deflate compression. Because many browsers include a
		// buggy deflate compression algorithm, prefer gzip over deflate if both
		// are acceptable. TODO: include support for brotli algorithm: br.
		if encodingAlgorithm = "gzip"; strings.Contains(acceptableEncodings, encodingAlgorithm) {
			newWriteCloser = gzip.NewWriter(w)
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using gzip: %s", err), http.StatusInternalServerError)
				}
			}()
		} else if encodingAlgorithm = "deflate"; strings.Contains(acceptableEncodings, encodingAlgorithm) {
			var err error
			newWriteCloser, err = flate.NewWriter(w, flate.DefaultCompression)
			if err != nil {
				// This should never happen, but if cannot create a new deflate
				// writer, then ignore the Accept-Encoding header and send the
				// unchanged request to the downstream handler.
				next.ServeHTTP(w, r)
				return
			}
			defer func() {
				if err := newWriteCloser.Close(); err != nil {
					Error(w, fmt.Sprintf("cannot compress stream using deflate: %s", err), http.StatusInternalServerError)
				}
			}()
		} else {
			// Upstream requests a compression algorithms that is not
			// supported. Ignore the Accept-Encoding header and send the
			// unchanged request to the downstream handler.
			next.ServeHTTP(w, r)
			return
		}

		// Delete the Accept-Encoding header from the request to prevent
		// downstream handler from seeing it and possibly also compressing data,
		// resulting in a payload that needs to be decompressed twice.
		r.Header.Del(requestHeader)

		// Set the response headers accordingly.
		w.Header().Set(responseHeader, encodingAlgorithm)
		w.Header().Set("Vary", responseHeader)

		// Have the downstream handler service this request, writing the
		// response to our compression writer.
		next.ServeHTTP(compressionResponseWriter{ResponseWriter: w, compressionWriter: newWriteCloser}, r)
	})
}
