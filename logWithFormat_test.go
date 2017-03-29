package gohm_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/karrick/gohm"
)

const apacheTimeFormat = "02/Jan/2006:15:04:05 MST"

func TestWithFormatStatusEscapedCharacters(t *testing.T) {
	format := "\\{client-ip\\}"
	logBitmask := uint32(1 | 2 | 4 | 8 | 16)
	logOutput := new(bytes.Buffer)

	handler := gohm.LogStatusBitmaskWithFormat(format, &logBitmask, logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	req := httptest.NewRequest("GET", "/some/url", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := logOutput.String(), "{client-ip}\n"; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestWithFormatStatic(t *testing.T) {
	format := "{client} {client-ip} {client-port} - \"{method} {uri} {proto}\" {status} {bytes}"
	logBitmask := uint32(1 | 2 | 4 | 8 | 16)
	logOutput := new(bytes.Buffer)

	handler := gohm.LogStatusBitmaskWithFormat(format, &logBitmask, logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	req := httptest.NewRequest("GET", "/some/url", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

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

func TestWithFormatIgnoresInvalidTokens(t *testing.T) {
	format := "This is an {invalid-token} with a {status} after it"
	logBitmask := uint32(1 | 2 | 4 | 8 | 16)
	logOutput := new(bytes.Buffer)

	handler := gohm.LogStatusBitmaskWithFormat(format, &logBitmask, logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	req := httptest.NewRequest("GET", "/some/url", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

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

func TestWithFormatDynamic(t *testing.T) {
	format := "{begin-epoch} {end-epoch} {begin} {begin-iso8601} {end} {end-iso8601} {duration}"
	logBitmask := uint32(1 | 2 | 4 | 8 | 16)
	logOutput := new(bytes.Buffer)

	handler := gohm.LogStatusBitmaskWithFormat(format, &logBitmask, logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}))

	beforeTime := time.Now() //.Round(time.Second)

	req := httptest.NewRequest("GET", "/some/url", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	afterTime := time.Now() //.Round(time.Second)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if actual, expected := resp.StatusCode, http.StatusForbidden; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

	if actual, expected := string(body), "403 Forbidden: some error\n"; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}

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

func BenchmarkWithCustomFormatter(b *testing.B) {
	format := "{client-ip} [{end}] \"{method} {uri} {proto}\" {status} {bytes} {duration}"
	logBitmask := uint32(1 | 2 | 4 | 8 | 16)
	logOutput := new(bytes.Buffer)

	handler := gohm.LogStatusBitmaskWithFormat(format, &logBitmask, logOutput, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/some/url", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		logOutput.Reset()
	}
}
