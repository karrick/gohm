package gohm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

func TestTimeoutHandlerNoTimeout(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	handler := gohm.TimeoutHandler(time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestTimeoutHandlerTimeout(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	handler := gohm.TimeoutHandler(time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusServiceUnavailable; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "took too long to process request"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
