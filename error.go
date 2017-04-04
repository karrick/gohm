package gohm

import (
	"net/http"
	"strconv"
)

// Error formats and emits the specified error message text and status code information to the
// http.ResponseWriter, to be consumed by the client of the service.  This particular helper
// function has nothing to do with emitting log messages on the server side, and only creates a
// response for the client.  However, if a handler that invokes gohm.Error is wrapped with logging
// functionality by gohm.New, then gohm will also emit a sensible log message based on the specified
// status code and message text.  Typically handlers will call this method prior to invoking return
// to return to whichever handler invoked it.
//
//	// example function which guards downstream handlers to ensure only HTTP GET method used
//	// to access resource.
//	func onlyGet(next http.Handler) http.Handler {
//			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				if r.Method != "GET" {
//					gohm.Error(w, r.Method, http.StatusMethodNotAllowed)
//					// 405 Method Not Allowed: POST
//					return
//				}
//				next.ServeHTTP(w, r)
//			})
//	}
func Error(w http.ResponseWriter, text string, code int) {
	fullText := strconv.Itoa(code) + " " + http.StatusText(code)
	if text != "" {
		fullText += ": " + text
	}
	http.Error(w, fullText, code)
}
