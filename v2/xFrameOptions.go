package gohm

import "net/http"

// XFrameOptions sets the X-Frame-Options response header to the specified
// value, then serves the request by the specified next handler.
//
// The X-Frame-Options HTTP response header is frequently used to block against
// clickjacking attacks. See https://tools.ietf.org/html/rfc7034 for more
// information.
//
// someHandler = gohm.XFrameOptions("SAMEORIGIN", someHandler)
func XFrameOptions(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", value)
		next.ServeHTTP(w, r)
	})
}
