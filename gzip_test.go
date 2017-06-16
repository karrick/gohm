package gohm_test

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

func TestGzipUncompressed(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	handler := gohm.WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Header().Get("Content-Encoding"), ""; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestGzipCompressed(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	response := "{pi:3.14159265}"

	handler := gohm.WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Header().Get("Content-Encoding"), "gzip"; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	gz, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	blob, err := ioutil.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(blob), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
