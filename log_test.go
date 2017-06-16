package gohm_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

func TestLogAllWithoutError(t *testing.T) {
	logOutput := new(bytes.Buffer)

	response := "something interesting"

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}), gohm.Config{LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusOK); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogAllWithError(t *testing.T) {
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusConflict)
	}), gohm.Config{LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusConflict); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

// then set log errors, and try both error and not error

func TestLogErrorsWhenWriteHeaderErrorStatus(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusForbidden); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogErrorsWithError(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), fmt.Sprintf(" %d ", http.StatusForbidden); !strings.Contains(actual, expected) {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogErrorsWithoutError(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no error
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), ""; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

const apacheTimeFormat = "02/Jan/2006:15:04:05 -0700"

func TestLogWithFormatStatusEscapedCharacters(t *testing.T) {
	format := "\\{client-ip\\}"
	logBitmask := gohm.LogStatusAll
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput, LogFormat: format})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	if actual, expected := logOutput.String(), "{client-ip}\n"; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogWithFormatStatic(t *testing.T) {
	format := "{client} {client-ip} {client-port} - \"{method} {uri} {proto}\" {status} {bytes}"
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogFormat: format, LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	// hardcoded test request remote address
	client := req.RemoteAddr
	clientIP := client
	clientPort := client
	if colon := strings.LastIndex(client, ":"); colon != -1 {
		clientIP = client[:colon]
		clientPort = client[colon+1:]
	}

	expected := fmt.Sprintf("%s %s %s - \"GET /some/url HTTP/1.1\" %d 26\n", client, clientIP, clientPort, http.StatusForbidden)
	if actual := logOutput.String(); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestLogWithFormatIgnoresInvalidTokens(t *testing.T) {
	format := "This is an {invalid-token} with a {status} after it"
	logBitmask := gohm.LogStatusAll
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogFormat: format, LogWriter: logOutput})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	expected := fmt.Sprintf("This is an {invalid-token} with a %d after it\n", http.StatusForbidden)
	if actual := logOutput.String(); actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func timeFromEpochString(t *testing.T, value string) time.Time {
	epoch, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return time.Unix(int64(epoch), 0)
}

func TestLogWithFormatDynamic(t *testing.T) {
	format := "{begin-epoch} {end-epoch} {begin} {begin-iso8601} {end} {end-iso8601} {duration}"
	logBitmask := gohm.LogStatusAll
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogFormat: format, LogWriter: logOutput})

	beforeTime := time.Now()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/url", nil)

	handler.ServeHTTP(rr, req)

	afterTime := time.Now()

	// first, grab the begin-epoch, and compute the other begin values
	actual := logOutput.String()

	indexFirstSpace := strings.IndexByte(actual, ' ')
	beginString := actual[:indexFirstSpace]

	beginTime := timeFromEpochString(t, beginString)
	if beginTime.Before(beforeTime.Truncate(time.Second)) {
		t.Errorf("Begin: %v; Before: %v", beginTime, beforeTime)
	}
	if beginTime.After(afterTime) {
		t.Errorf("Begin: %v; After: %v", beginTime, afterTime)
	}

	// first, grab the end-epoch, and compute the other end values
	indexSecondSpace := indexFirstSpace + strings.IndexByte(actual[indexFirstSpace+1:], ' ')
	endString := actual[indexFirstSpace+1 : indexSecondSpace+1]
	endTime := timeFromEpochString(t, endString)
	if endTime.Before(beforeTime.Truncate(time.Second)) {
		t.Errorf("End: %v; Before: %v", endTime, beforeTime)
	}
	if endTime.After(afterTime) {
		t.Errorf("End: %v; After: %v", endTime, afterTime)
	}

	if actual, expected := actual[len(actual)-1:], "\n"; actual != expected {
		t.Errorf("Actual: %#v; #Expected: %#v", actual, expected)
	}

	indexFinalSpace := strings.LastIndexByte(actual, ' ')
	durationString := actual[indexFinalSpace+1 : len(actual)-1]

	// to check duration, let's just ensure we can parse it as a float, and it's less than the span duration
	durationFloat, err := strconv.ParseFloat(durationString, 64)
	if err != nil {
		t.Errorf("Actual: %#v; Expected: %#v", err, nil)
	}
	durationMilliseconds := afterTime.Sub(beforeTime).Nanoseconds() / 1000
	if int64(durationFloat*1000000) > durationMilliseconds {
		t.Errorf("durationFloat: %v; durationMilliseconds: %v", durationFloat, durationMilliseconds)
	}

	expected := fmt.Sprintf("%s %s %s %s %s %s %s\n", beginString, endString,
		beginTime.UTC().Format(apacheTimeFormat),
		beginTime.UTC().Format(time.RFC3339),
		endTime.UTC().Format(apacheTimeFormat),
		endTime.UTC().Format(time.RFC3339),
		durationString)

	if actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func BenchmarkWithLogsElided(b *testing.B) {
	logBitmask := uint32(gohm.LogStatus4xx | gohm.LogStatus5xx) // only errors
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this does not error, so nothing ought to be logged
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/some/url", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		logOutput.Reset()
	}
}

func BenchmarkWithLogs(b *testing.B) {
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // error class forces log entry
	}), gohm.Config{LogWriter: logOutput})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/some/url", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		logOutput.Reset()
	}
}
