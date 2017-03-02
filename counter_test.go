package gohm_test

import (
	"expvar"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karrick/gohm"
)

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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(1); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
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

	if actual, expected := counter.Value(), int64(0); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
