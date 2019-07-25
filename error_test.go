package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

// when error called, status updated, client receives message
func TestWhenErrorInvoked(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := resp.StatusCode, http.StatusForbidden; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := string(body), "403 Forbidden: some error\n"; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}
