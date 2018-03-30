package gohm

import (
	"path"
	"testing"
)

func TestShiftPath(t *testing.T) {
	cases := map[string][]string{
		"":             []string{"", "/"},
		".":            []string{"", "/"},
		"/foo":         []string{"foo", "/"},
		"/foo/":        []string{"foo", "/"},
		"/foo/bar":     []string{"foo", "/bar"},
		"/foo/bar/baz": []string{"foo", "/bar/baz"},
		"foo":          []string{"foo", "/"},
		"foo/":         []string{"foo", "/"},
		"foo/bar":      []string{"foo", "/bar"},
		"foo/bar/":     []string{"foo", "/bar"},
		"foo/bar/baz":  []string{"foo", "/bar/baz"},
	}

	for k, v := range cases {
		k = path.Clean("/" + k)[1:]
		car, cdr := shiftPath(k)
		if want, got := v[0], car; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", k, want, got)
		}
		if want, got := v[1], cdr; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", k, want, got)
		}
	}
}
