package gohm

import (
	"strings"
)

// shiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. The first return value will never
// contain a slash, while the second return value will always be a path starting
// with a slash.
//
// Inspired by:
//     https://blog.merovius.de/2017/06/18/how-not-to-use-an-http-router.html
//
// Expects paths to be initially cleaned:
//     p = path.Clean("/" + p)[1:]
//     head, tail := shiftPath(p)
func shiftPath(p string) (string, string) {
	if len(p) == 0 {
		return p, "/"
	}
	i := strings.Index(p[1:], "/") // start searching after expected first character as slash
	switch i {
	case -1: // not found
		// fmt.Fprintf(os.Stderr, "not found: %q\n", p)
		return p[1:], "/"
	case 0: // double-slash at start
		// fmt.Fprintf(os.Stderr, "double slash: %q\n", p)
		return p[2:], "/"
	default: // other
		// fmt.Fprintf(os.Stderr, "other: %q\n", p)
		return p[1 : i+1], p[i+1:]
	}
}
