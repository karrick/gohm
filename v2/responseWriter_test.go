package gohm_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/karrick/gobp"
	"github.com/karrick/gohm/v2"
)

func buildResponse() string {
	var response string
	// For short tests, use a small buffer; for regular tests, use a larger
	// buffer.
	size := 512
	if !testing.Short() {
		size = 32768 + rand.Intn(65536)
	}
	for i := 0; i < size; i++ {
		response += "a"
	}
	return response
}

func TestResponseWriter(t *testing.T) {
	response := buildResponse()
	bp := new(gobp.Pool)

	t.Run("without buffer pool", func(t *testing.T) {
		testResponseWriter(t, response, nil)
	})

	t.Run("with buffer pool", func(t *testing.T) {
		testResponseWriter(t, response, bp)
	})
}

func testResponseWriter(tb testing.TB, response string, bp gohm.BytesBufferPool) {
	statusCode := http.StatusCreated

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("foo", "foo1")
		w.Header().Add("foo", "foo2")
		w.Header().Add("bar", "bar1")
		w.Header().Add("bar", "bar2")
		w.WriteHeader(statusCode)
		n, err := w.Write([]byte(response))
		if got, want := n, len(response); got != want {
			tb.Errorf("GOT: %v; WANT: %v", got, want)
		}
		if got, want := err, error(nil); got != want {
			tb.Errorf("GOT: %v; WANT: %v", got, want)
		}
	}), gohm.Config{BufPool: bp})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()

	if got, want := resp.StatusCode, statusCode; got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tb.Fatal(err)
	}
	if got, want := string(body), response; got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}

	// created sorted list of keys
	var keys []string
	for key := range resp.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// ensure list of keys match
	if got, want := fmt.Sprintf("%s", keys), "[Bar Foo]"; got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}

	// for every got key, ensure values match
	for key, values := range resp.Header {
		sort.Strings(values)
		var want string
		switch key {
		case "Bar":
			want = "[bar1 bar2]"
		case "Foo":
			want = "[foo1 foo2]"
		}
		if got := fmt.Sprintf("%s", values); got != want {
			tb.Errorf("Key: %q; Got: %#v; Want: %#v", key, got, want)
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
	if got, want := resp.StatusCode, http.StatusForbidden; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	// but when only invoke WriteHeader, nothing gets written to client
	if got, want := string(body), ""; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func BenchmarkResponseWriter(b *testing.B) {
	bp := new(gobp.Pool)
	response := buildResponse()

	b.ResetTimer()

	b.Run("without buffer pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testResponseWriter(b, response, nil)
		}
	})

	b.Run("with buffer pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testResponseWriter(b, response, bp)
		}
	})
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
