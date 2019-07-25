package gohm_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

func TestResponseWriter(t *testing.T) {
	status := http.StatusCreated
	response := "some response"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		header := w.Header()
		header.Add("foo", "foo1")
		header.Add("foo", "foo2")
		header.Add("bar", "bar1")
		header.Add("bar", "bar2")
		w.Write([]byte(response))
	}), gohm.Config{})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()

	if got, want := resp.StatusCode, status; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	// created sorted list of keys
	var keys []string
	for key := range resp.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// ensure list of keys match
	if got, want := fmt.Sprintf("%s", keys), "[Bar Foo]"; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	// for every key, ensure values match
	for key, values := range resp.Header {
		sort.Strings(values)
		var want string
		switch key {
		case "Bar":
			want = "[bar1 bar2]"
		case "Foo":
			want = "[foo1 foo2]"
		}
		if got := fmt.Sprintf("%s", values); got != want {
			t.Errorf("KEY: %q; GOT: %v; WANT: %v", key, got, want)
		}
	}
}

func TestResponseWriterWhenWriteHeaderErrorStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), gohm.Config{})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// status code ought to have been sent to client
	if got, want := resp.StatusCode, http.StatusForbidden; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	// but when only invoke WriteHeader, nothing gets written to client
	if got, want := string(body), ""; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func BenchmarkWithoutResponseWriter(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkWithDefaultResponseWriter(b *testing.B) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), gohm.Config{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkWithFullResponseWriter(b *testing.B) {
	var counters gohm.Counters

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), gohm.Config{
		Counters:  &counters,
		LogWriter: ioutil.Discard,
		Timeout:   time.Second,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}
