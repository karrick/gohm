package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

func BenchmarkFullWithoutSink(b *testing.B) {
	var counters gohm.Counters
	logBitmask := gohm.LogStatusAll

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	// h = gohm.WithGzip(h)                  // gzip response if client accepts gzip encoding
	h = gohm.WithTimeout(time.Second, h)  // immediately return when downstream hasn't replied within specified time
	h = gohm.WithCloseNotifier(h)         // immediately return when the client disconnects
	h = gohm.ConvertPanicsToErrors(h)     // when downstream panics, convert to 500
	h = gohm.StatusCounters(&counters, h) // update counter stats for 1xx, 2xx, 3xx, 4xx, 5xx, and all queries
	h = gohm.LogStatusBitmaskWithFormat(gohm.DefaultLogFormat, &logBitmask, ioutil.Discard, h)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		h.ServeHTTP(rr, req)
	}
}
func BenchmarkEmptySink(b *testing.B) {
	handler := gohm.Sink(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), new(gohm.Config))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkFullSink(b *testing.B) {
	var counters gohm.Counters

	handler := gohm.Sink(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), &gohm.Config{
		CloseNotifier:         true,
		ConvertPanicsToErrors: true,
		Counters:              &counters,
		LogBitmask:            gohm.LogStatusAll,
		LogFormat:             gohm.DefaultLogFormat,
		LogWriter:             ioutil.Discard,
		Timeout:               time.Second,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}
