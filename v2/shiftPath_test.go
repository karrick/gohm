package gohm_test

import (
	"testing"

	"github.com/karrick/gohm/v2"
)

func TestShiftPath(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"", []string{"", "/"}},                        // invalid URL because empty string
		{".", []string{"", "/"}},                       // invalid URL because does not start with slash
		{"/", []string{"", "/"}},                       // valid URL
		{"//", []string{"", "/"}},                      // invalid URL because of double slash
		{"/foo", []string{"foo", "/"}},                 // valid URL
		{"/foo/", []string{"foo", "/"}},                // valid URL
		{"/foo/bar", []string{"foo", "/bar"}},          // valid URL
		{"/foo/bar/baz", []string{"foo", "/bar/baz"}},  // valid URL
		{"/foo/bar/baz/", []string{"foo", "/bar/baz"}}, // valid URL
		{"foo", []string{"oo", "/"}},                   // invalid URL because does not start with slash
	}

	for _, c := range cases {
		first, rest := gohm.ShiftPath(c.in)
		if want, got := c.out[0], first; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
		if want, got := c.out[1], rest; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
	}
}
