package gohm

import (
	"net/http"
	"time"
)

// TimeoutHandler returns a new http.Handler that modifies creates a new http.Request instance with
// the specified timeout set via context.
func TimeoutHandler(timeout time.Duration, next http.Handler) http.Handler {
	return http.TimeoutHandler(next, timeout, "took too long to process request")

	// TODO: Write a custom handler to cancel the inflight request on timeout. Collect metrics for this.

	// return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	// 	defer cancel()
	// 	next.ServeHTTP(w, r.WithContext(ctx))
	// })
}
