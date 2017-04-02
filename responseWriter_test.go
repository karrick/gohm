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

	handler := gohm.WithTimeout(time.Minute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		header := w.Header()
		header.Add("foo", "foo1")
		header.Add("foo", "foo2")
		header.Add("bar", "bar1")
		header.Add("bar", "bar2")
		w.Write([]byte(response))
	}))

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

func TestResponseWriterWhenPanic(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.WithTimeout(5*time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test error")
	}))

	rr := httptest.NewRecorder()

	panicked := false
	served := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
			close(served)
		}()
		handler.ServeHTTP(rr, req)
	}()

	<-served

	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := panicked, true; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	// NOTE: Cannot verify resp.StatusCode because httptest.ResponseRecorder initializes StatusCode to http.StatusOK
	// if actual, expected := resp.StatusCode, http.StatusInternalServerError; actual != expected {
	// 	t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	// }
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

func BenchmarkWithCloseNotifier(b *testing.B) {
	handler := gohm.WithCloseNotifier(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}
