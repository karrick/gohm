package gohm_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm/v2"
)

func TestPrefix(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		cases := []struct {
			in  string
			out []string
		}{
			{"/foo", []string{"foo", "/"}},
			{"/foo/", []string{"foo", "/"}},
			{"/foo/bar", []string{"foo", "/bar"}},
			{"/foo/bar/", []string{"foo", "/bar"}},
			{"/foo/bar/baz", []string{"foo", "/bar/baz"}},
			{"/foo/bar/baz/", []string{"foo", "/bar/baz"}},
		}

		for _, c := range cases {
			request := httptest.NewRequest("POST", c.in, ioutil.NopCloser(bytes.NewReader([]byte("line1\nline2\n"))))
			gotString := gohm.Prefix(request)
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
	})

	t.Run("example", func(t *testing.T) {
		v1Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(fmt.Sprintf("v1 %s\r\n", r.URL.Path)))
		})

		v2Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(fmt.Sprintf("v2 %s\r\n", r.URL.Path)))
		})

		apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch prefix := gohm.Prefix(r); prefix {
			case "v1":
				v1Handler(w, r)
			case "v2":
				v2Handler(w, r)
			default:
				http.Error(w, prefix, http.StatusNotFound)
			}
		})

		t.Run("v1", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/v1/foo", nil)

			apiHandler.ServeHTTP(recorder, request)

			resp := recorder.Result()
			body, err := readAllThenClose(resp.Body)
			ensureError(t, err)

			if got, want := resp.StatusCode, http.StatusOK; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
			if got, want := string(body), "v1 /foo\r\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})

		t.Run("v2", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/v2/foo", nil)

			apiHandler.ServeHTTP(recorder, request)

			resp := recorder.Result()
			body, err := readAllThenClose(resp.Body)
			ensureError(t, err)

			if got, want := resp.StatusCode, http.StatusOK; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
			if got, want := string(body), "v2 /foo\r\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})

		t.Run("v3", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/v3/foo", nil)

			apiHandler.ServeHTTP(recorder, request)

			resp := recorder.Result()
			body, err := readAllThenClose(resp.Body)
			ensureError(t, err)

			if got, want := resp.StatusCode, http.StatusNotFound; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
			if got, want := string(body), "v3\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})
	})
}

func readAllThenClose(rc io.ReadCloser) ([]byte, error) {
	buf, rerr := ioutil.ReadAll(rc)
	cerr := rc.Close() // always close regardless of read error
	if rerr != nil {
		return buf, rerr // Read error has more context than Close error
	}
	return buf, cerr
}
