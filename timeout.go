package gohm

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// WithTimeout returns a new http.Handler that creates a watchdog goroutine to detect when the
// timeout has expired.  It also modifies the request to add a context timeout, because while not
// all handlers use context and respect context timeouts, it's likely that more and more will over
// time as context becomes more popular.
//
// Unlike when using http.TimeoutHandler, if a downstream http.Handler panics, this handler will
// catch that panic in the other goroutine and re-play it in the primary goroutine, allowing
// upstream handlers to catch the panic if desired.  Panics may be caught by the
// gohm.ConvertPanicsToErrors handler when placed upstream of this handler.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithTimeout(10 * time.Second, someHandler))
func WithTimeout(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a responseWriter to pass to next.ServeHTTP and collect downstream
		// handler's response to query.  It will eventually be used to flush to the client,
		// assuming neither the handler panics, nor the client connection is detected to be
		// closed.
		rw := &responseWriter{ResponseWriter: w}

		// Create a couple of channels to detect one of 3 ways to exit this handler.
		serverCompleted := make(chan struct{})
		serverPanicked := make(chan string, 1)
		timedOut := make(chan struct{})

		// Watchdog goroutine sits and waits for the timeout to expire and trigger required
		// actions if it does.
		go func() {
			time.Sleep(timeout)
			close(timedOut)
		}()

		// While not all handlers use context and would respect timeout, it's likely that
		// more and more will over time as context becomes more popular.  Even though this
		// handler will handle the timeout, we modify the context so any context-aware
		// handlers downstream will get the signal when the timeout has elapsed.
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)

		// We must invoke downstream handler in separate goroutine in order to ensure this
		// handler only responds to one of the three events below, whichever event takes
		// place first.
		go serveWithPanicProtection(rw, r, next, serverCompleted, serverPanicked)

		// Wait for the first of either of 3 events:
		//   * serveComplete: the next.ServeHTTP method completed normally (possibly even
		//     with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and panicked
		//     instead with a text message.
		//   * context is done: triggered when timeout has been exceeded.
		select {

		case <-serverCompleted:
			if err := rw.flush(); err != nil {
				Error(w, fmt.Sprintf("cannot flush response writer: %s", err), http.StatusInternalServerError)
			}

		case text := <-serverPanicked:
			// Error(w, text, http.StatusInternalServerError)

			// NOTE: While this could simply emit the error message here, right now it
			// re-panics from this goroutine, effectively capturing and replaying the
			// panic from the downstream handler that took place in a different
			// goroutine.
			panic(text) // do not need to tell downstream to cancel, because it already panicked.

		case <-timedOut:
			// timeout watchdog routine triggered
			Error(w, "took too long to process request", http.StatusServiceUnavailable) // 503 (this is what http.TimeoutHandler returns)

		case <-ctx.Done():
			// the context was canceled; where ctx.Err() will say why
			Error(w, ctx.Err().Error(), http.StatusServiceUnavailable) // 503 (this is what http.TimeoutHandler returns)

		}
	})
}
