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

	"github.com/karrick/gohm/v2"
)

func TestLogAllWithoutError(t *testing.T) {
	logOutput := new(bytes.Buffer)
	response := "something interesting"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}), gohm.Config{LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), fmt.Sprintf(" %d ", http.StatusOK); !strings.Contains(got, want) {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestLogAllWithError(t *testing.T) {
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusConflict)
	}), gohm.Config{LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), fmt.Sprintf(" %d ", http.StatusConflict); !strings.Contains(got, want) {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

// then set log errors, and try both error and not error

func TestLogErrorsWhenWriteHeaderErrorStatus(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), fmt.Sprintf(" %d ", http.StatusForbidden); !strings.Contains(got, want) {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestLogErrorsWithError(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), fmt.Sprintf(" %d ", http.StatusForbidden); !strings.Contains(got, want) {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestLogErrorsWithoutError(t *testing.T) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no error
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), ""; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

const apacheTimeFormat = "02/Jan/2006:15:04:05 -0700"

func TestLogWithFormatStatusEscapedCharacters(t *testing.T) {
	format := "\\{client-ip\\}"
	logBitmask := gohm.LogStatusAll
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput, LogFormat: format})

	handler.ServeHTTP(recorder, request)

	if got, want := logOutput.String(), "{client-ip}\n"; got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestLogWithFormatStatic(t *testing.T) {
	format := "{client} {client-ip} {client-port} - \"{method} {uri} {proto}\" {status} {bytes}"

	t.Run("ipv4", func(t *testing.T) {
		logOutput := new(bytes.Buffer)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		request.RemoteAddr = "1.2.3.4:1234"

		handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gohm.Error(w, "some error", http.StatusForbidden)
		}), gohm.Config{LogFormat: format, LogWriter: logOutput})

		handler.ServeHTTP(recorder, request)

		want := fmt.Sprintf("1.2.3.4:1234 1.2.3.4 1234 - \"GET /some/url HTTP/1.1\" %d 26\n", http.StatusForbidden)
		if got := logOutput.String(); got != want {
			t.Fatalf("\nGOT:\n\t%#v\nWant:\n\t%#v", got, want)
		}
	})

	t.Run("ipv6", func(t *testing.T) {
		logOutput := new(bytes.Buffer)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		request.RemoteAddr = "[::1]:1234"

		handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gohm.Error(w, "some error", http.StatusForbidden)
		}), gohm.Config{LogFormat: format, LogWriter: logOutput})

		handler.ServeHTTP(recorder, request)

		want := fmt.Sprintf("[::1]:1234 ::1 1234 - \"GET /some/url HTTP/1.1\" %d 26\n", http.StatusForbidden)
		if got := logOutput.String(); got != want {
			t.Fatalf("\nGOT:\n\t%#v\nWant:\n\t%#v", got, want)
		}
	})

}

func TestLogWithFormatIgnoresInvalidTokens(t *testing.T) {
	format := "This is an {invalid-token} with a {status} after it"
	logBitmask := gohm.LogStatusAll
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogFormat: format, LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	want := fmt.Sprintf("This is an {invalid-token} with a %d after it\n", http.StatusForbidden)
	if got := logOutput.String(); got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
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

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogBitmask: &logBitmask, LogFormat: format, LogWriter: logOutput})

	beforeTime := time.Now()
	handler.ServeHTTP(recorder, request)
	afterTime := time.Now()

	// first, grab the begin-epoch, and compute the other begin values
	got := logOutput.String()

	indexFirstSpace := strings.IndexByte(got, ' ')
	beginString := got[:indexFirstSpace]

	beginTime := timeFromEpochString(t, beginString)
	if beginTime.Before(beforeTime.Truncate(time.Second)) {
		t.Errorf("Begin: %v; Before: %v", beginTime, beforeTime)
	}
	if beginTime.After(afterTime) {
		t.Errorf("Begin: %v; After: %v", beginTime, afterTime)
	}

	// first, grab the end-epoch, and compute the other end values
	indexSecondSpace := indexFirstSpace + strings.IndexByte(got[indexFirstSpace+1:], ' ')
	endString := got[indexFirstSpace+1 : indexSecondSpace+1]
	endTime := timeFromEpochString(t, endString)
	if endTime.Before(beforeTime.Truncate(time.Second)) {
		t.Errorf("End: %v; Before: %v", endTime, beforeTime)
	}
	if endTime.After(afterTime) {
		t.Errorf("End: %v; After: %v", endTime, afterTime)
	}

	if got, want := got[len(got)-1:], "\n"; got != want {
		t.Errorf("Got: %#v; #Want: %#v", got, want)
	}

	indexFinalSpace := strings.LastIndexByte(got, ' ')
	durationString := got[indexFinalSpace+1 : len(got)-1]

	// to check duration, let's just ensure we can parse it as a float, and it's less than the span duration
	durationFloat, err := strconv.ParseFloat(durationString, 64)
	if err != nil {
		t.Errorf("GOT: %v; WANT: %v", err, nil)
	}
	durationMilliseconds := afterTime.Sub(beforeTime).Nanoseconds() / 1000
	if int64(durationFloat*1000000) > durationMilliseconds {
		t.Errorf("durationFloat: %v; durationMilliseconds: %v", durationFloat, durationMilliseconds)
	}

	want := fmt.Sprintf("%s %s %s %s %s %s %s\n", beginString, endString,
		beginTime.UTC().Format(apacheTimeFormat),
		beginTime.UTC().Format(time.RFC3339),
		endTime.UTC().Format(apacheTimeFormat),
		endTime.UTC().Format(time.RFC3339),
		durationString)

	if got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func TestLogWithFormatEmoji(t *testing.T) {
	format := "😛 {status} 😊" // log format starts and ends with a multi-byte rune
	logOutput := new(bytes.Buffer)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/some/url", nil)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gohm.Error(w, "some error", http.StatusForbidden)
	}), gohm.Config{LogFormat: format, LogWriter: logOutput})

	handler.ServeHTTP(recorder, request)

	got := logOutput.String()
	want := fmt.Sprintf("😛 %d 😊\n", http.StatusForbidden)
	if got != want {
		t.Fatalf("GOT: %v; WANT: %v", got, want)
	}
}

func BenchmarkWithLogsElided(b *testing.B) {
	logBitmask := gohm.LogStatusErrors
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this does not error, so nothing ought to be logged
	}), gohm.Config{LogBitmask: &logBitmask, LogWriter: logOutput})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
		logOutput.Reset()
	}
}

func BenchmarkWithLogs(b *testing.B) {
	logOutput := new(bytes.Buffer)

	handler := gohm.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // error class forces log entry
	}), gohm.Config{LogWriter: logOutput})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/some/url", nil)
		handler.ServeHTTP(recorder, request)
		logOutput.Reset()
	}
}
