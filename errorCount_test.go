package gohm_test

import (
	"expvar"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

func TestErrorCountHandlerNoStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	errorCounter := expvar.NewInt("error-count-no-status")

	handler := gohm.ErrorCountHandler(errorCounter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if actual, expected := errorCounter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestErrorCountHandlerStatusOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	errorCounter := expvar.NewInt("error-count-status-ok")

	handler := gohm.ErrorCountHandler(errorCounter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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

	if actual, expected := errorCounter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestErrorCountHandlerError(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	errorCounter := expvar.NewInt("error-count-with-error")

	handler := gohm.ErrorCountHandler(errorCounter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := errorCounter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
