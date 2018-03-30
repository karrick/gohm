package gohm

import (
	"fmt"
	"net/http"
)

// ConvertPanicsToErrors returns a new http.Handler that catches all panics that may be caused by
// the specified http.Handler, and responds with an appropriate http status code and message.
//
//	mux := http.NewServeMux()
//	mux.Handle("/example/path", gohm.ConvertPanicsToErrors(someHandler))
func ConvertPanicsToErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCompleted := make(chan struct{})
		serverPanicked := make(chan string, 1)

		go serveWithPanicProtection(w, r, next, serverCompleted, serverPanicked)

		// Wait for the first of either of 2 events:
		//   * serveComplete: the next.ServeHTTP method completed normally (possibly even
		//     with an erroneous status code).
		//   * servePanicked: the next.ServeHTTP method failed to complete, and panicked
		//     instead with a text message.
		select {
		case <-serverCompleted:
			// no action required
		case text := <-serverPanicked:
			Error(w, text, http.StatusInternalServerError)
		}
	})
}

// Attempt to serve the query by calling the original handler, next.  Normally the handler completes
// ServeHTTP, and this will close the completed channel.  If the ServeHTTP method panics, then the
// panicked error text is sent to the paniched channel.
func serveWithPanicProtection(w http.ResponseWriter, r *http.Request, next http.Handler, completed chan struct{}, panicked chan<- string) {
	defer func() {
		if r := recover(); r != nil {
			var text string
			switch t := r.(type) {
			case error:
				text = t.Error()
			case string:
				text = t
			default:
				text = fmt.Sprintf("%v", r)
			}
			panicked <- text
		}
	}()
	next.ServeHTTP(w, r)
	// will not get here if above line panics
	close(completed)
}
