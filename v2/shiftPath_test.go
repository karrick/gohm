package gohm

import (
	"testing"
)

func TestShiftPath(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"", []string{"", "/"}},   // invalid
		{".", []string{"", "/"}},  // invalid
		{"/", []string{"", "/"}},  // legal
		{"//", []string{"", "/"}}, // invalid
		{"/foo", []string{"foo", "/"}},
		{"/foo/", []string{"foo", "/"}},
		{"/foo/bar", []string{"foo", "/bar"}},
		{"/foo/bar/baz", []string{"foo", "/bar/baz"}},
		{"/foo/bar/baz/", []string{"foo", "/bar/baz/"}},
		{"foo", []string{"oo", "/"}}, // invalid
	}

	for _, c := range cases {
		first, rest := shiftPath(c.in)
		if want, got := c.out[0], first; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
		if want, got := c.out[1], rest; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
	}
}
