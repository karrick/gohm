package gohm_test

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm/v2"
)

func TestGzipUncompressed(t *testing.T) {
	response := "{pi:3.14159265}"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Header().Get("Content-Encoding"), ""; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Body.String(), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func TestGzipCompressed(t *testing.T) {
	response := "{pi:3.14159265}"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	handler := gohm.WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Header().Get("Content-Encoding"), "gzip"; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	iorc, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := iorc.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	blob, err := ioutil.ReadAll(iorc)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(blob), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func TestCompressionUncompressed(t *testing.T) {
	response := "{pi:3.14159265}"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.WithCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Header().Get("Content-Encoding"), ""; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Body.String(), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func TestCompressionGzipPreferredOverDeflate(t *testing.T) {
	response := "{pi:3.14159265}"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)
	request.Header.Set("Accept-Encoding", "deflate, gzip, br")

	handler := gohm.WithCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if compression := r.Header.Get("Accept-Encoding"); compression != "" {
			gohm.Error(w, fmt.Sprintf("ought to have removed `Accept-Encoding` request header: %q", compression), http.StatusBadRequest)
			return
		}
		w.Write([]byte(response))
	}))

	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Header().Get("Content-Encoding"), "gzip"; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	iorc, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := iorc.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	blob, err := ioutil.ReadAll(iorc)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(blob), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func TestCompressionDeflateWorks(t *testing.T) {
	response := "{pi:3.14159265}"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)
	request.Header.Set("Accept-Encoding", "br, deflate")

	handler := gohm.WithCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if compression := r.Header.Get("Accept-Encoding"); compression != "" {
			gohm.Error(w, fmt.Sprintf("ought to have removed `Accept-Encoding` request header: %q", compression), http.StatusBadRequest)
			return
		}
		w.Write([]byte(response))
	}))

	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}

	if got, want := recorder.Header().Get("Content-Encoding"), "deflate"; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	iorc := flate.NewReader(recorder.Body)
	defer func() {
		if err := iorc.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	blob, err := ioutil.ReadAll(iorc)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(blob), response; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}
