package gohm

import "strings"

// ShiftPath splits off the first component of p. The first return value will
// never contain a slash, while the second return value will always start with a
// slash, followed by all remaining path components, with the final slash
// removed when remaining path components are returned.
//
// Example:
//
//     "/foo/bar/baz"  -> "foo", "/bar/baz"
//     "/foo/bar/baz/" -> "foo", "/bar/baz"
//
// Inspired by:
//     https://blog.merovius.de/2017/06/18/how-not-to-use-an-http-router.html
//
// Expects paths to be initially cleaned:
//     p = path.Clean("/" + p)[1:]
//     head, tail := shiftPath(p)
func ShiftPath(p string) (string, string) {
	l := len(p)
	if l == 0 {
		return p, "/"
	}
	i := strings.Index(p[1:], "/") // start searching after expected first character as slash
	switch i {
	case -1: // not found
		return p[1:], "/"
	case 0: // double-slash at start
		return p[2:], "/"
	default: // other
		if l-i <= 2 || p[l-1] != '/' {
			return p[1 : i+1], p[i+1:] // no final slash or not enough characters
		} else {
			return p[1 : i+1], p[i+1 : l-1] // final slash removed
		}
	}
}
