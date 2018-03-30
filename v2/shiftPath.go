package gohm

import "strings"

// shiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
//
// Inspired by:
//     https://blog.merovius.de/2017/06/18/how-not-to-use-an-http-router.html
//
// Expects paths to be initially cleaned:
//     p = path.Clean("/" + p)[1:]
//     head, tail := shiftPath(p)
func shiftPath(p string) (string, string) {
	if i := strings.Index(p, "/"); i >= 0 {
		return p[:i], p[i:]
	}
	return p, "/"
}
