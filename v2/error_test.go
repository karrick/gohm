package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/karrick/gohm/v2"
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

	if got, want := string(body), "some error"; !strings.Contains(got, want) {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}
