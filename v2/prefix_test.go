package gohm

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestPrefix(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{"/foo", []string{"foo", "/"}},
		{"/foo/", []string{"foo", "/"}},
		{"/foo/bar", []string{"foo", "/bar"}},
		{"/foo/bar/baz", []string{"foo", "/bar/baz"}},
	}

	for _, c := range cases {
		request := httptest.NewRequest("POST", c.in, ioutil.NopCloser(bytes.NewReader([]byte("line1\nline2\n"))))
		gotString := prefix(request)
		if want, got := c.out[0], gotString; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
		if want, got := c.out[1], request.URL.Path; want != got {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
		// Ensure body is still readable; not checking other structure fields.
		gotBody, err := readAllThenClose(request.Body)
		if want, got := (error)(nil), err; got != want {
			t.Errorf("%q: WANT: %v; GOT: %v", c.in, want, got)
		}
		if want, got := "line1\nline2\n", string(gotBody); got != want {
			t.Errorf("%q: WANT: %q; GOT: %q", c.in, want, got)
		}
	}
}

func readAllThenClose(rc io.ReadCloser) ([]byte, error) {
	buf, rerr := ioutil.ReadAll(rc)
	cerr := rc.Close() // always close regardless of read error
	if rerr != nil {
		return buf, rerr // Read error has more context than Close error
	}
	return buf, cerr
}
