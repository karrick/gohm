package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/karrick/gohm/v2"
)

func TestTimeout(t *testing.T) {
	const response = "{pi:3.14159265}"

	t.Run("before", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)

		handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(response))
		}), gohm.Config{Timeout: time.Second})

		handler.ServeHTTP(recorder, request)

		resp := recorder.Result()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := string(body), response; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})

	t.Run("after", func(t *testing.T) {
		response := "{pi:3.14159265}"

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)

		handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second)
			w.Write([]byte(response))
		}), gohm.Config{Timeout: 5 * time.Millisecond})

		handler.ServeHTTP(recorder, request)

		resp := recorder.Result()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := resp.StatusCode, http.StatusServiceUnavailable; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := string(body), "context deadline exceeded"; !strings.Contains(got, want) {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})
}

func BenchmarkWithTimeout(b *testing.B) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't bother exceeding timeout
	}), gohm.Config{Timeout: time.Second})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}
