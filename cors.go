package gohm

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CORSConfig holds parameters for configuring a CORSHandler.
type CORSConfig struct {
	// OriginsFilter is a regular expression that acts as a filter against the
	// "Origin" header value for pre-flight checks.
	OriginsFilter *regexp.Regexp

	// AllowHeaders is a list of HTTP header names which are allowed to be sent
	// to this handler.
	AllowHeaders []string

	// AllowMethods is a list of HTTP method names which are allowed for this
	// handler.
	AllowMethods []string

	// MaxAgeSeconds is the number of seconds used to fill the
	// "Access-Control-Max-Age" header in pre-flight check responses.
	MaxAgeSeconds int
}

// CORSHandler returns a handler that responds to OPTIONS request so that CORS
// requests from an origin that matches the specified allowed origins regular
// expression are permitted, while other origins are denied. If a request origin
// matches the specified regular expression, the handler responds with the
// specified allowOriginResponse value in the "Access-Control-Allow-Origin" HTTP
// response header.
func CORSHandler(config CORSConfig, next http.Handler) http.Handler {
	// By definition a CORS handler will respond to the OPTIONS method, so
	// include that method if not already specified.
	config.AllowMethods = sortAndMaybeInsertString("OPTIONS", config.AllowMethods)
	allowedMethods := strings.Join(config.AllowMethods, ", ")

	// Most browser frameworks also send "X-Requested-With" header, and we want
	// to allow such requests.
	config.AllowHeaders = sortAndMaybeInsertString("X-Requested-With", config.AllowHeaders)
	allowHeaders := strings.Join(config.AllowHeaders, ", ")

	maxAge := strconv.Itoa(config.MaxAgeSeconds)

	return AllowedMethodsHandler(config.AllowMethods, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When Cross Origin Resource Sharing (CORS) request arrives, the
		// browser submits an "Origin" header that specifies where the request
		// came from. This handler will deny requests that do not match the
		// specified regular expression.

		if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" {
			// The browser has requested an Origin check, which may be either a
			// so called "simple-request," or a pre-flight request.
			if !config.OriginsFilter.MatchString(requestOrigin) {
				Error(w, fmt.Sprintf("origin domain not permitted: %q", requestOrigin), http.StatusForbidden)
				return
			}

			// All responses, not just those for pre-flight checks, require
			// "Access-Control-Allow-Origin" header to handle so-called "simple
			// requests," which do not require a pre-flight check by the
			// browser, yet the browser still expects the response's headers to
			// include this value.
			w.Header().Set("Access-Control-Allow-Origin", "*") // requestOrigin)

			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Max-Age", maxAge)
				w.Header().Set("Allow", allowedMethods)

				// During pre-flight checks, browser also submits the following
				// header to specify what method it would like to use.
				requestMethod := r.Header.Get("Access-Control-Request-Method")
				i := sort.SearchStrings(config.AllowMethods, requestMethod)
				if i == len(config.AllowMethods) || config.AllowMethods[i] != requestMethod {
					// Requested method is not on the list of allowed methods.
					Error(w, requestMethod, http.StatusMethodNotAllowed)
					// fall through to return
				}

				return // nothing further to do for this OPTIONS handler
			}

			// fall through to next handler
		}

		next.ServeHTTP(w, r)
	}))
}

// AllowedMethodsHandler returns a handler that only permits specified request
// methods, and responds with an error message when request method is not a
// member of the sorted list of allowed methods.
func AllowedMethodsHandler(allowedMethods []string, next http.Handler) http.Handler {
	sortedAllowedMethods := make([]string, len(allowedMethods))
	copy(sortedAllowedMethods, allowedMethods)
	sort.Strings(sortedAllowedMethods)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := sort.SearchStrings(sortedAllowedMethods, r.Method)
		if i == len(sortedAllowedMethods) || sortedAllowedMethods[i] != r.Method {
			Error(w, r.Method, http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sortAndMaybeInsertString(s string, a []string) []string {
	if len(a) == 0 {
		return append(a, s)
	}

	sort.Strings(a)

	i := sort.SearchStrings(a, s)

	if i < len(a) && a[i] == s {
		return a // return slice when string already present
	}

	// Without two copies and mandatory allocation, insert string into array at
	// index.
	a = append(a, a[len(a)-1])
	copy(a[i+1:], a[i:len(a)-1])
	a[i] = s

	return a
}
