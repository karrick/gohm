package gohm

import (
	"net/http"
	"time"
)

// WithTimeout returns a new http.Handler that modifies creates a new http.Request instance with
// the specified timeout set via context.
//
// 	mux := http.NewServeMux()
// 	mux.Handle("/example/path", gohm.WithTimeout(30 * time.Second, onlyGet(someHandler)))
//
// *NOTE:* When both the WithTimeout and the ConvertPanicsToErrors are used, the WithTimeout ought
// to wrap the ConvertPanicsToErrors.  This is because timeout handlers in Go are generally
// implemented using a separate go routine, and the panic could occur in an alternate go routine and
// not get caught by the ConvertPanicsToErrors.
func WithTimeout(timeout time.Duration, next http.Handler) http.Handler {
	// Using the timeout handling provided by the standard library.
	return http.TimeoutHandler(next, timeout, "took too long to process request")

	// TODO: Write a custom handler to cancel the inflight request on timeout. Collect metrics for this.

	// return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	// 	defer cancel()
	// 	next.ServeHTTP(w, r.WithContext(ctx))
	// })
}
