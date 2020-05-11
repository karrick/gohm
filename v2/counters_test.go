package gohm_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm/v2"
)

func testCounter(t *testing.T, status int) gohm.Counters {
	t.Helper()
	var counters gohm.Counters
	const responseBody = "some response\r\n"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			// To ensure status code set even when next handler does not
			// explicitly set it, we omit setting header when StatusOK is
			// provided, and below we invoke this test function for both
			// http.StatusOK and another 2xx status code.
			w.WriteHeader(status)
		}
		w.Write([]byte(responseBody))
	}), gohm.Config{Counters: &counters})

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := response.StatusCode, status; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := string(body), responseBody; got != want {
		t.Errorf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.GetAll(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	return counters
}

func TestStatusCounters1xx(t *testing.T) {
	counters := testCounter(t, http.StatusContinue) // 100

	if got, want := counters.Get1xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	// get and reset counter
	if got, want := counters.GetAndReset1xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestStatusCounters2xxWithoutHandlerWritingStatus(t *testing.T) {
	counters := testCounter(t, http.StatusOK) // 200

	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestStatusCounters2xx(t *testing.T) {
	counters := testCounter(t, http.StatusCreated) // 201

	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	// get and reset counter
	if got, want := counters.GetAndReset2xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestStatusCounters3xx(t *testing.T) {
	counters := testCounter(t, http.StatusFound) // 302

	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	// get and reset counter
	if got, want := counters.GetAndReset3xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestStatusCounters4xx(t *testing.T) {
	counters := testCounter(t, http.StatusForbidden) // 403

	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	// get and reset counter
	if got, want := counters.GetAndReset4xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestStatusCounters5xx(t *testing.T) {
	counters := testCounter(t, http.StatusGatewayTimeout) // 504

	if got, want := counters.Get1xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get2xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get3xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get4xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}

	// get and reset counter
	if got, want := counters.GetAndReset5xx(), uint64(1); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := counters.Get5xx(), uint64(0); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func BenchmarkWithCounters(b *testing.B) {
	var counters gohm.Counters

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), gohm.Config{Counters: &counters})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
	}
}
