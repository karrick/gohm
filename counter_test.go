package gohm_test

import (
	"expvar"
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
	handler := gohm.StatusCounters(&counters, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			// To ensure status code set even when next handler does not explicitly set
			// it, we omit setting header when StatusOK is provided, and below we invoke
			// this test function for both http.StatusOK and another 2xx status code.
			w.WriteHeader(status)
		}
		w.Write([]byte(response))
	}))

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

// TESTS FOR DEPRECATED FUNCTIONALITY

func TestStatusAllCounterHandlerOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("all-ok")
	response := "gimme more!!!"
	status := http.StatusOK // 200

	handler := gohm.StatusAllCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatusAllCounterHandlerNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("all-not-found")
	response := "what?"
	status := http.StatusNotFound // 404

	handler := gohm.StatusAllCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus1xxCounterHandlerIncrementCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("1xx-continue")
	response := "gimme more!!!"
	status := http.StatusContinue // 100

	handler := gohm.Status1xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus1xxCounterHandlerIgnoreCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("1xx-created")
	response := "record created"
	status := http.StatusCreated // 201

	handler := gohm.Status1xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus2xxCounterHandlerWriteHeaderElided(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("2xx-write-header-elided")
	response := "elided"
	status := http.StatusOK

	handler := gohm.Status2xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus2xxCounterHandlerIncrementCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("2xx-accepted")
	response := "gimme more!!!"
	status := http.StatusAccepted // 202

	handler := gohm.Status2xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus2xxCounterHandlerIgnoreCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("2xx-processing")
	response := "record created"
	status := http.StatusProcessing // 102

	handler := gohm.Status2xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus3xxCounterHandlerIncrementCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("3xx-found")
	response := "I found it"
	status := http.StatusFound // 302

	handler := gohm.Status3xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus3xxCounterHandlerIgnoreCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("3xx-already-reported")
	response := "record created"
	status := http.StatusAlreadyReported // 208

	handler := gohm.Status3xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus4xxCounterHandlerIncrementCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("4xx-bad-request")
	response := "gimme more!!!"
	status := http.StatusBadRequest // 400

	handler := gohm.Status4xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus4xxCounterHandlerIgnoreCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("4xx-see-other")
	response := "record created"
	status := http.StatusSeeOther // 303

	handler := gohm.Status4xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus5xxCounterHandlerIncrementCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("5xx-internal-server-error")
	response := "gimme more!!!"
	status := http.StatusInternalServerError // 500

	handler := gohm.Status5xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestStatus5xxCounterHandlerIgnoreCounter(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	counter := expvar.NewInt("5xx-unauthorized")
	response := "record created"
	status := http.StatusUnauthorized // 401

	handler := gohm.Status5xxCounter(counter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := string(body), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := resp.StatusCode, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
