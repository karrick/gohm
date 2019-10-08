package gohm

import (
	"crypto/tls"
	"mime/multipart"
	"net/http"
	"net/url"
)

// prefix strips the prefix from the start of the request and returns it, along
// with a new request that has the prefix removed. When the request is for the
// empty URL, it returns an empty string for the prefix, and a pointer to the
// original request.
func prefix(r *http.Request) (string, *http.Request) {
	if r.URL.Path == "" {
		return "", r
	}

	r2 := new(http.Request)
	*r2 = *r

	r2.URL = new(url.URL)
	*r2.URL = *r.URL

	if r.MultipartForm != nil {
		r2.MultipartForm = new(multipart.Form)
		*r2.MultipartForm = *r.MultipartForm
	}

	if r.TLS != nil {
		r2.TLS = new(tls.ConnectionState)
		*r2.TLS = *r.TLS
	}

	if r.Body != nil {
		r2.Body = r.Body
	}

	var prefix string
	prefix, r2.URL.Path = shiftPath(r.URL.Path[1:])

	return prefix, r2
}
