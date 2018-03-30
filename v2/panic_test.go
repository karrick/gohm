package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/karrick/gohm/v2"
)

func TestAllowPanicsFalse(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test error")
	}), gohm.Config{})

	var panicked bool
	served := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
			close(served)
		}()
		handler.ServeHTTP(recorder, request)
	}()

	<-served

	resp := recorder.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := panicked, false; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, http.StatusInternalServerError; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := string(body), "500 Internal Server Error"; !strings.Contains(actual, expected) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestAllowPanicsTrue(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test error")
	}), gohm.Config{AllowPanics: true})

	var panicked bool
	served := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
			close(served)
		}()
		handler.ServeHTTP(recorder, request)
	}()

	<-served

	resp := recorder.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := panicked, true; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	// NOTE: Cannot verify resp.StatusCode because httptest.ResponseRecorder
	// initializes StatusCode to http.StatusOK if not written, even though it is
	// never set.
	if actual, expected := string(body), ""; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
