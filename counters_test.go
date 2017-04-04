package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

func test(t *testing.T, status int) gohm.Counters {
	var counters gohm.Counters
	response := "some response"
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)
	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			// To ensure status code set even when next handler does not explicitly set
			// it, we omit setting header when StatusOK is provided, and below we invoke
			// this test function for both http.StatusOK and another 2xx status code.
			w.WriteHeader(status)
		}
		w.Write([]byte(response))
	}), gohm.Config{Counters: &counters})

	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.GetAll(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	return counters
}

func TestStatusCounters1xx(t *testing.T) {
	counters := test(t, http.StatusContinue) // 100

	if actual, expected := counters.Get1xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusCounters2xxWithoutHandlerWritingStatus(t *testing.T) {
	counters := test(t, http.StatusOK) // 200

	if actual, expected := counters.Get1xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusCounters2xx(t *testing.T) {
	counters := test(t, http.StatusCreated) // 201

	if actual, expected := counters.Get1xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusCounters3xx(t *testing.T) {
	counters := test(t, http.StatusFound) // 302

	if actual, expected := counters.Get1xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusCounters4xx(t *testing.T) {
	counters := test(t, http.StatusForbidden) // 403

	if actual, expected := counters.Get1xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusCounters5xx(t *testing.T) {
	counters := test(t, http.StatusGatewayTimeout) // 504

	if actual, expected := counters.Get1xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get2xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get3xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get4xx(), uint64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counters.Get5xx(), uint64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func BenchmarkWithCounters(b *testing.B) {
	var counters gohm.Counters

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), gohm.Config{Counters: &counters})

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(rr, req)
	}
}
