package gohm

import (
	"context"
	"fmt"
	"net/http"
)

// WithCloseNotifier returns a new http.Handler that attempts to detect when the client has closed
// the connection, and if it does so, immediately returns with an appropriate error message to be
// logged, while sending a signal to context-aware downstream handlers.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.WithCloseNotifier(someHandler))
func WithCloseNotifier(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a responseWriter to pass to next.ServeHTTP and collect downstream
		// handler's response to query.  It will eventually be used to flush to the client,
		// assuming neither the handler panics, nor the client connection is detected to be
		// closed.
		rw := &responseWriter{ResponseWriter: w}

		// Create a couple of channels to detect one of 3 ways to exit this handler.
		clientDisconnected := make(chan struct{})
		serverCompleted := make(chan struct{})
		serverPanicked := make(chan string, 1)

		// Not all http.ResponseHandlers implement http.CloseNotifier.  If the
		// http.ResponseHandler we were given does, then we can use it to detect when the
		// client has closed its connection socket.  If the http.ResponseWriter does not
		// implement http.CloseNotifier, there same overhead applies, but the downstream
		// handlers will still work correctly, however, this handler simply will not detect
		// when the client has closed the connection.
		if notifier, ok := w.(http.CloseNotifier); ok {
			receivingBlocksUntilRemoteClosed := notifier.CloseNotify()
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			r = r.WithContext(ctx)

			// Watchdog goroutine sits and waits for the client to possibly close the
			// connection, and trigger required actions if it does.
			go func() {
				<-receivingBlocksUntilRemoteClosed
				// When here, the remote has closed connection.

				// Tell downstream it may stop trying to serve the request.  Many
				// handlers still ignore context cancellations, but we do what we
				// can.
				cancel()

				// Terminate this handler, and if logger attached upstream, let's
				// throw in a descriptive server log message
				close(clientDisconnected)
			}()
		}

		// We must invoke downstream handler in separate goroutine in order to ensure this
		// handler only responds to one of the three events below, whichever event takes
		// place first.
		go serveWithPanicProtection(rw, r, next, serverCompleted, serverPanicked)

		// Wait for the first of either of 3 events:
		//   * serveComplete: the next.ServeHTTP method completed normally (possibly even
		//     with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and panicked
		//     instead with a text message.
		//   * clientDisconnected: the client has disconnected the HTTP connection prior to
		//     either of the above conditions.
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

		case <-clientDisconnected:
			Error(w, "cannot serve to disconnected client", http.StatusRequestTimeout) // 408 (or should we use another?)

		}
	})
}
