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
	const responseBody = "test panic"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(responseBody)
	}), gohm.Config{})

	var panicked bool
	served := make(chan struct{})

	// Invoke the hander inside a go routine with panic protection, to make sure
	// that gohm catches the panic itself, and responds to http.ResponseWriter
	// properly.
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

	if got, want := panicked, false; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := resp.StatusCode, http.StatusInternalServerError; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := string(body), responseBody; !strings.Contains(got, want) {
		t.Errorf("GOT: %v; WANT: %v", got, want)
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

	if got, want := panicked, true; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	// NOTE: Cannot verify resp.StatusCode because httptest.ResponseRecorder
	// initializes StatusCode to http.StatusOK if not written, even though it is
	// never set.
	if got, want := string(body), ""; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}
