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

func TestBeforeTimeout(t *testing.T) {
	response := "{pi:3.14159265}"

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

	if actual, expected := resp.StatusCode, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestAfterTimeout(t *testing.T) {
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

	if actual, expected := resp.StatusCode, http.StatusServiceUnavailable; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "503 Service Unavailable"; !strings.Contains(actual, expected) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
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
