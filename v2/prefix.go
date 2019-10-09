package gohm

import "net/http"

// prefix strips the prefix from the start of the request's path and returns it,
// modifying the request's path by removing the stripped prefix. When the
// request path is for the empty URL, it returns an empty string and does not
// modify the request.
func prefix(r *http.Request) (prefix string) {
	if r.URL.Path != "" {
		prefix, r.URL.Path = shiftPath(r.URL.Path)
		return
	}
	return r.URL.Path
}
