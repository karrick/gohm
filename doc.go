/*
Package gohm is a tiny Go library with HTTP middleware functions.

gohm provides a small collection of middleware functions to be used when creating a HTTP micro
service written in Go.

One function in particular, gohm.Error, is not used as HTTP middleware, but as a helper for emitting
a sensible error message back to the HTTP client when the HTTP request could not be fulfilled.  It
emits a text response beginning with the status code, the human friendly status message, followed by
an optional text message.  It is meant to be a drop in replacement for http.Error message, that
formats the error message in a more conventional way to include the status code and message.

	// Example function which guards downstream handlers to ensure only HTTP GET method used
	// to access resource.
	func onlyGet(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				gohm.Error(w, r.Method, http.StatusMethodNotAllowed)
				// 405 Method Not Allowed: POST
				return
			}
			next.ServeHTTP(w, r)
		})
	}

All the other gohm functions are HTTP middleware functions, designed to wrap any HTTP handler,
composing some functionality around it.  They can be interchanged and used with other HTTP
middleware functions providing those functions adhere to the http.Handler interface and have a
ServeHTTP(http.ResponseHandler, *http.Request) method.

    mux := http.NewServeMux()
    var h http.HandlerFunc = someHandler
    h = gohm.WithGzip(h)
    h = gohm.ConvertPanicsToErrors(h)
    h = gohm.WithTimeout(globalTimeout, h)
    h = gohm.LogErrors(os.Stderr, h)
    mux.Handle("/static/", h)

*NOTE:* When both the WithTimeout and the ConvertPanicsToErrors are used, the WithTimeout ought to
wrap the ConvertPanicsToErrors.  This is because timeout handlers in Go are generally implemented
using a separate go routine, and the panic could occur in an alternate go routine and not get caught
by the ConvertPanicsToErrors.

*/
package gohm
