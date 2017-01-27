package gohm

import (
	"fmt"
	"net/http"
)

// PanicHandler returns a new http.Handler that catches a panic caused by the specified
// http.Handler, and responds with an appropriate http status code and message.
func PanicHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				var text string
				switch t := r.(type) {
				case error:
					text = t.Error()
				case string:

				default:
					text = fmt.Sprintf("%v", r)
				}
				Error(w, text, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
