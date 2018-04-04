package gohm

import (
	"net/http"
	"strings"
)

// DefaultHandler serves the specified file when request is for an empty path,
// "/", but serves a 404 Not Found for all other requests.
//
// When most users first visit a website, they visit it with an empty URL
// path. This handler responds to such a request with the contents of the
// specified file. However, the standard library's HTTP multiplexer type,
// http.ServeMux, will cause all requests that have an URL path that do not
// match any other registered handler to match against "/", causing invalid URL
// path requests to be given the index page rather than a 404 Not Found. This
// handler corrects that behavior by differentiating whether the request URL
// path is "/". If so it serves the contents of the specified file; otherwise a
// 404 Not Found is returned to the user.
//
//   	http.Handle("/", gohm.DefaultHandler(filepath.Join(staticPath, "index.html")))
func DefaultHandler(pathOfIndexFile string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Client requests bare path => serve index file.
			http.ServeFile(w, r, pathOfIndexFile)
			return
		}
		// Client requests path that failed to match any other registered
		// handlers => serve 404 Not Found.
		Error(w, r.URL.Path, http.StatusNotFound)
	})
}

// ForbidDirectories prevents clients from probing file servers.
//
// ForbidDirectories will respond to all requests that have a "/" suffix with
// 403 Forbidden, but will forward all requests without the "/" suffix to the
// specified next handler.
//
// The standard library's http.FileServer will enumerate a directory's contents
// when a client requests a directory. This function's purpose is to prevent
// clients from probing a static file server to see its resources by attempting
// to query directories.
//
//     http.Handle("/static/", gohm.ForbidDirectories(gohm.StaticHandler("/static/", staticPath)))
func ForbidDirectories(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			Error(w, r.URL.Path, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// StaticHandler serves static files, preventing clients from probing file
// servers.
//
// The specified virtual root will be stripped from the prefix, such that when
// "$virtualRoot/foo/bar" is requested, "$fileSystemRoot/foo/bar" will be
// served.
//
// This handler will respond to all requests that have a "/" suffix with 403
// Forbidden, but will attempt to serve all requests without the "/" suffix by
// serving the corresponding file.
//
// The standard library's http.FileServer will enumerate a directory's contents
// when a client requests a directory. This function's purpose is to prevent
// clients from probing a static file server to see its resources by attempting
// to query directories.
//
//     	http.Handle("/static/", gohm.StaticHandler("/static/", staticPath))
func StaticHandler(virtualRoot, fileSystemRoot string) http.Handler {
	fileServingHandler := http.FileServer(http.Dir(fileSystemRoot))
	return ForbidDirectories(http.StripPrefix(virtualRoot, fileServingHandler))
}

// StaticHandlerWithoutProbingProtection serves static files, and when a
// directory is requested, will serve a representation of the directory's
// contents.
//
// The specified virtual root will be stripped from the prefix, such that when
// "$virtualRoot/foo/bar" is requested, "$fileSystemRoot/foo/bar" will be
// served.
//
// Please use the StaticHandler function rather than this function, unless your
// application specifically benefits from clients probing your file server's
// contents.
func StaticHandlerWithoutProbingProtection(virtualRoot, fileSystemRoot string) http.Handler {
	fileServingHandler := http.FileServer(http.Dir(fileSystemRoot))
	return http.StripPrefix(virtualRoot, fileServingHandler)
}
