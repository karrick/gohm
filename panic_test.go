package gohm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

func TestConvertPanicsToErrorsWhenNoPanic(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "nothing special"
	status := http.StatusOK

	handler := gohm.ConvertPanicsToErrors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestConvertPanicsToErrorsWhenPanic(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.ConvertPanicsToErrors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("some horrible event took place")
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusInternalServerError; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "500 Internal Server Error: some horrible event took place\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
