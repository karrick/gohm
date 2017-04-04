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
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
