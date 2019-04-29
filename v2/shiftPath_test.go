package gohm

import (
	"path"
	"testing"
)

func TestShiftPath(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"", []string{"", "/"}},
		{".", []string{"", "/"}},
		{"/foo", []string{"foo", "/"}},
		{"/foo/", []string{"foo", "/"}},
		{"/foo/bar", []string{"foo", "/bar"}},
		{"/foo/bar/baz", []string{"foo", "/bar/baz"}},
		{"foo", []string{"foo", "/"}},
		{"foo/", []string{"foo", "/"}},
		{"foo/bar", []string{"foo", "/bar"}},
		{"foo/bar/", []string{"foo", "/bar"}},
		{"foo/bar/baz", []string{"foo", "/bar/baz"}},
	}

	for _, c := range cases {
		k := path.Clean("/" + c.in)[1:]
		car, cdr := shiftPath(k)
		if want, got := c.out[0], car; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", k, want, got)
		}
		if want, got := c.out[1], cdr; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", k, want, got)
		}
	}
}
