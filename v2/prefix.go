package gohm

import "net/http"

// prefix strips the prefix from the start of the request and returns it, and
// modifies the request to remove the stripped prefix. When the request is for
// the empty URL, it returns an empty string and does not modify the request.
func prefix(r *http.Request) (prefix string) {
	if r.URL.Path == "" {
		return r.URL.Path
	}
	prefix, r.URL.Path = shiftPath(r.URL.Path)
	return
}
