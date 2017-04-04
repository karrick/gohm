package gohm

import (
	"fmt"
	"net/http"
)

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
