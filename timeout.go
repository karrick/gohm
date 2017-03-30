package gohm

import (
	"net/http"
	"time"
)

// WithTimeout returns a new http.Handler that modifies creates a new http.Request instance with the
// specified timeout set via context.
//
// *NOTE:* When both the WithTimeout and the ConvertPanicsToErrors are used, the WithTimeout ought
// to wrap the ConvertPanicsToErrors.  This is because WithTimeout does not itself implement the
// timeout, but requests the net/http library to do so, which implements timeout handling using a
// separate go routine.  When a panic occurs in a separate go routine it will not get caught by
// ConvertPanicsToErrors.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithTimeout(30 * time.Second, onlyGet(someHandler)))
func WithTimeout(timeout time.Duration, next http.Handler) http.Handler {
	// Using the timeout handling provided by the standard library.
	return http.TimeoutHandler(next, timeout, "took too long to process request")

	// TODO: Consider Writing a custom handler to cancel the inflight request on timeout.  Collect metrics for this.

	// return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	// 	defer cancel()
	// 	next.ServeHTTP(w, r.WithContext(ctx))
	// })
}
