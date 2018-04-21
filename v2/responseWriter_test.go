package gohm_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/karrick/gobp"
	"github.com/karrick/gohm/v2"
)

func TestResponseWriter(t *testing.T) {
	status := http.StatusCreated
	response := "some response"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		header := w.Header()
		header.Add("foo", "foo1")
		header.Add("foo", "foo2")
		header.Add("bar", "bar1")
		header.Add("bar", "bar2")
		w.Write([]byte(response))
	}), gohm.Config{})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()

	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	// created sorted list of keys
	var keys []string
	for key := range resp.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// ensure list of keys match
	if actual, expected := fmt.Sprintf("%s", keys), "[Bar Foo]"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	// for every actual key, ensure values match
	for key, values := range resp.Header {
		sort.Strings(values)
		var expected string
		switch key {
		case "Bar":
			expected = "[bar1 bar2]"
		case "Foo":
			expected = "[foo1 foo2]"
		}
		if actual := fmt.Sprintf("%s", values); actual != expected {
			t.Errorf("Key: %q; Actual: %#v; Expected: %#v", key, actual, expected)
		}
	}
}

func TestResponseWriterWhenWriteHeaderErrorStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), gohm.Config{})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// status code ought to have been sent to client
	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	// but when only invoke WriteHeader, nothing gets written to client
	if actual, expected := string(body), ""; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func BenchmarkWithoutResponseWriter(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkWithDefaultResponseWriter(b *testing.B) {
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), gohm.Config{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkWithFullResponseWriter(b *testing.B) {
	var counters gohm.Counters

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), gohm.Config{
		Counters:  &counters,
		LogWriter: ioutil.Discard,
		Timeout:   time.Second,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}

func TestEscrowTestBodyNoErrors(t *testing.T) {
	const payload = "flubber"

	var handlerReadPayload []byte

	// origingalHandler is the http.Handler that normally handles a particular
	// request. It needs to be able to read then close the request body.
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReadPayload, _ = ioutil.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
	})

	var callbackBegin, callbackEnd time.Time
	var callbackBody []byte
	var callbackStatus int

	// wrapper wraps the original http.Handler, providing a callback that is
	// invoked upon completion of the original request http.Handler.
	wrapper := gohm.New(originalHandler, gohm.Config{
		BufPool: new(gobp.Pool),
		Callback: func(stats *gohm.Statistics) {
			callbackBegin = stats.RequestBegin
			callbackBody = stats.RequestBody
			callbackStatus = stats.ResponseStatus
			callbackEnd = stats.ResponseEnd
		},
		EscrowReader: true,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/some/url", bytes.NewReader([]byte(payload)))
	request.Header.Set("Content-Length", strconv.Itoa(len(payload)))

	before := time.Now()
	wrapper.ServeHTTP(recorder, request)
	after := time.Now()

	if got, want := string(handlerReadPayload), payload; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := string(callbackBody), payload; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := callbackStatus, http.StatusBadRequest; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := before.Before(callbackBegin), true; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := callbackBegin.Before(callbackEnd), true; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := callbackEnd.Before(after), true; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}
