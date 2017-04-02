package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

func TestWithTimeoutWhenNoTimeout(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	handler := gohm.WithTimeout(time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestWithTimeoutWhenTimeout(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	handler := gohm.WithTimeout(10*time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusServiceUnavailable; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "503 Service Unavailable"; !strings.Contains(actual, expected) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestWithTimeoutWhenPanic(t *testing.T) {
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
