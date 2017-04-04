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
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		header := w.Header()
		header.Add("foo", "foo1")
		header.Add("foo", "foo2")
		header.Add("bar", "bar1")
		header.Add("bar", "bar2")
		w.Write([]byte(response))
	}), gohm.Config{})

	handler.ServeHTTP(rr, req)

	resp := rr.Result()

	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	// created sorted list of keys
	var keys []string
	for key := range resp.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// ensure list of keys match
	if actual, expected := fmt.Sprintf("%s", keys), "[Bar Foo]"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	// for every actual key, ensure values match
	for key, values := range resp.Header {
		sort.Strings(values)
		var expected string
		switch key {
		case "Bar":
			expected = "[bar1 bar2]"
		case "Foo":
			expected = "[foo1 foo2]"
		}
		if actual := fmt.Sprintf("%s", values); actual != expected {
			t.Errorf("Key: %q; Actual: %#v; Expected: %#v", key, actual, expected)
		}
	}
}

func TestResponseWriterWhenWriteHeaderErrorStatus(t *testing.T) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), gohm.Config{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// status code ought to have been sent to client
	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	// but when only invoke WriteHeader, nothing gets written to client
	if actual, expected := string(body), ""; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func BenchmarkWithoutResponseWriter(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkWithDefaultResponseWriter(b *testing.B) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), gohm.Config{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
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
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}
