package gohm

import (
	"net/http"
	"strings"
)

// DefaultHandler serves the specified file when for the empty URI path, "/",
// but serves a 404 Not Found for all other requests.
func DefaultHandler(pathOfIndexFile string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, pathOfIndexFile)
			return
		}
		Error(w, r.URL.Path, http.StatusNotFound)
	})
}

// ForbidDirectories will respond to request with a "/" suffix with a HTTP
// forbidden status code.
//
//     h := gohm.ForbidDirectories(gohm.StaticHandler("/static/", "static"))
func ForbidDirectories(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			Error(w, r.URL.Path, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// StaticHandler serves static assets.
//
// The specified virtual root will be stripped from the prefix, such that when
// "$virtualRoot/foo/bar" is requested, "$fileSystemRoot/foo/bar" will be
// served. Request URL paths with a "/" suffix will not be served, but instead
// http.StatusForbidden will be returned.
func StaticHandler(virtualRoot, fileSystemRoot string) http.Handler {
	fileServingHandler := http.FileServer(http.Dir(fileSystemRoot))
	return http.StripPrefix(virtualRoot, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileServingHandler.ServeHTTP(w, r)
	}))
}
