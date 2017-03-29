package gohm

import (
	"fmt"
	"net/http"
)

// ConvertPanicsToErrors returns a new http.Handler that catches all panics that may be caused by
// the specified http.Handler, and responds with an appropriate http status code and message.
//
// 	mux := http.NewServeMux()
// 	mux.Handle("/example/path", gohm.ConvertPanicsToErrors(onlyGet(someHandler)))
//
// *NOTE:* When both the WithTimeout and the ConvertPanicsToErrors are used, the WithTimeout ought
// to wrap the ConvertPanicsToErrors.  This is because timeout handlers in Go are generally
// implemented using a separate go routine, and the panic could occur in an alternate go routine and
// not get caught by the ConvertPanicsToErrors.
func ConvertPanicsToErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				Error(w, text, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
