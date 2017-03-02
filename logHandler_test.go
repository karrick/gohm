package gohm_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/karrick/gohm"
)

func TestErrorLogHandlerNoStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	logOutput := new(bytes.Buffer)

	handler := gohm.ErrorLogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), ""; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestErrorLogHandlerStatusOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	response := "{pi:3.14159265}"

	logOutput := new(bytes.Buffer)

	handler := gohm.ErrorLogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), ""; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestErrorLogHandlerError(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	logOutput := new(bytes.Buffer)
	status := http.StatusForbidden

	handler := gohm.ErrorLogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", status)
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, status; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", status); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogHandlerError(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	logOutput := new(bytes.Buffer)

	handler := gohm.ErrorLogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusForbidden); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogHandlerNoError(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	logOutput := new(bytes.Buffer)

	response := "something interesting"

	handler := gohm.LogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusOK; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusOK); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogHandlerNonOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/some/url", nil)

	logOutput := new(bytes.Buffer)

	response := "something interesting"

	handler := gohm.LogHandler(logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
		w.Write([]byte(response))
	}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if actual, expected := rr.Code, http.StatusTemporaryRedirect; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := rr.Body.String(), response; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusTemporaryRedirect); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
